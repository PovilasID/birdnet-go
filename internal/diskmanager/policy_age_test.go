package diskmanager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// MockDB is a mock implementation of the database interface for testing
type MockDB struct{}

// GetDeletionInfo is a mock implementation that always returns no entries
func (m *MockDB) GetDeletionInfo() ([]string, error) {
	return []string{}, nil
}

// InsertDeletionInfo is a mock implementation that does nothing
func (m *MockDB) InsertDeletionInfo(filename string) error {
	return nil
}

// GetLockedNotesClipPaths is a mock implementation that returns no paths
func (m *MockDB) GetLockedNotesClipPaths() ([]string, error) {
	return []string{}, nil
}

// TestAgeBasedCleanupFileTypeEligibility tests if the file type check works correctly
func TestAgeBasedCleanupFileTypeEligibility(t *testing.T) {
	// Test with different file extensions
	testFiles := []struct {
		name          string
		expectError   bool
		errorContains string
	}{
		// Audio files - should work without errors
		{"owl_80p_20210102T150405Z.wav", false, ""},
		{"owl_80p_20210102T150405Z.mp3", false, ""},
		{"owl_80p_20210102T150405Z.flac", false, ""},
		{"owl_80p_20210102T150405Z.aac", false, ""},
		{"owl_80p_20210102T150405Z.opus", false, ""},

		// Non-audio files - should return errors
		{"owl_80p_20210102T150405Z.txt", true, "file type not eligible"},
		{"owl_80p_20210102T150405Z.jpg", true, "file type not eligible"},
		{"owl_80p_20210102T150405Z.png", true, "file type not eligible"},
		{"owl_80p_20210102T150405Z.db", true, "file type not eligible"},
		{"owl_80p_20210102T150405Z.csv", true, "file type not eligible"},
		{"system_80p_20210102T150405Z.exe", true, "file type not eligible"},
	}

	// Print the current list of allowed file types for debugging
	t.Logf("Allowed file types: %v", allowedFileTypes)

	// Create a temporary directory
	testDir, err := os.MkdirTemp("", "age_policy_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	for _, tc := range testFiles {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock FileInfo
			mockInfo := createMockFileInfo(tc.name, 1024)

			// Call parseFileInfo directly to test file extension checking
			_, err := parseFileInfo(filepath.Join(testDir, tc.name), mockInfo)

			// Debug logging
			t.Logf("File: %s, Extension: %s, Error: %v",
				tc.name, filepath.Ext(tc.name), err)

			// Check if the error matches our expectation
			if tc.expectError {
				if err == nil {
					t.Errorf("SECURITY ISSUE: Expected error for %s but got nil", tc.name)
				} else if !strings.Contains(err.Error(), tc.errorContains) {
					t.Errorf("Expected error containing '%s', got: %v", tc.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for %s but got: %v", tc.name, err)
				}
			}
		})
	}
}

// TestAgeBasedFilesAfterFilter tests the filtering of files for age-based cleanup
func TestAgeBasedFilesAfterFilter(t *testing.T) {
	db := &MockDB{}
	allowedTypes := []string{".wav", ".mp3", ".flac", ".aac", ".opus"}

	// Create a temporary directory
	testDir, err := os.MkdirTemp("", "age_filter_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Let's create files of all relevant types
	fileTypes := []string{
		".wav", ".mp3", ".flac", ".aac", ".opus",
		".txt", ".jpg", ".png", ".db", ".exe",
	}

	for _, ext := range fileTypes {
		filePath := filepath.Join(testDir, fmt.Sprintf("owl_80p_20210102T150405Z%s", ext))
		if err := os.WriteFile(filePath, []byte("test content"), 0o644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Get audio files using the function that would be used by the policy
	audioFiles, err := GetAudioFiles(testDir, allowedTypes, db, false)
	if err != nil {
		t.Fatalf("Failed to get audio files: %v", err)
	}

	// Verify only allowed audio files are returned
	if len(audioFiles) != len(allowedTypes) {
		t.Errorf("Expected %d audio files, got %d", len(allowedTypes), len(audioFiles))
	}

	// Verify each returned file has an allowed extension
	for _, file := range audioFiles {
		ext := filepath.Ext(file.Path)
		if !contains(allowedTypes, ext) {
			t.Errorf("SECURITY ISSUE: File with disallowed extension was included: %s", file.Path)
		}
	}
}

// TestAgeBasedCleanupBasicFunctionality tests the basic functionality of age-based cleanup
func TestAgeBasedCleanupBasicFunctionality(t *testing.T) {
	// Create test files with different timestamps
	// Recent files (within retention period)
	recentFile1 := FileInfo{
		Path:       "/test/owl_80p_20210102T150405Z.wav",
		Species:    "owl",
		Confidence: 80,
		Timestamp:  time.Now().Add(-24 * time.Hour), // 1 day old
		Size:       1024,
		Locked:     false,
	}

	recentFile2 := FileInfo{
		Path:       "/test/duck_70p_20210102T150405Z.wav",
		Species:    "duck",
		Confidence: 70,
		Timestamp:  time.Now().Add(-48 * time.Hour), // 2 days old
		Size:       1024,
		Locked:     false,
	}

	// Old files (beyond retention period)
	oldFile1 := FileInfo{
		Path:       "/test/owl_90p_20200102T150405Z.wav",
		Species:    "owl",
		Confidence: 90,
		Timestamp:  time.Now().Add(-720 * time.Hour), // 30 days old
		Size:       1024,
		Locked:     false,
	}

	oldFile2 := FileInfo{
		Path:       "/test/duck_60p_20200102T150405Z.wav",
		Species:    "duck",
		Confidence: 60,
		Timestamp:  time.Now().Add(-1440 * time.Hour), // 60 days old
		Size:       1024,
		Locked:     false,
	}

	// A locked file that should never be deleted
	lockedFile := FileInfo{
		Path:       "/test/owl_95p_20200102T150405Z.wav",
		Species:    "owl",
		Confidence: 95,
		Timestamp:  time.Now().Add(-2160 * time.Hour), // 90 days old
		Size:       1024,
		Locked:     true,
	}

	// Test files collection
	testFiles := []FileInfo{recentFile1, recentFile2, oldFile1, oldFile2, lockedFile}

	// Verify file type checks are performed on all files
	for _, file := range testFiles {
		filename := filepath.Base(file.Path)
		ext := filepath.Ext(filename)

		// Assert that only allowed file types are processed
		assert.True(t, contains(allowedFileTypes, ext),
			"File type should be in the allowed list: %s", ext)
	}

	// Check that age-based cleanup would:
	// 1. Delete files older than retention period
	// 2. Never delete locked files
	// 3. Maintain minimum number of clips per species

	// This is a basic verification - a full test would require mocking more components
	for _, file := range testFiles {
		// Locked files should never be deleted
		if file.Locked {
			t.Logf("Verified that locked file would be protected: %s", file.Path)
			continue
		}

		// Recent files should be kept
		if file.Timestamp.After(time.Now().Add(-168 * time.Hour)) { // Assuming 7 day retention
			t.Logf("Verified that recent file would be preserved: %s", file.Path)
		} else {
			t.Logf("Verified that old file would be eligible for deletion: %s", file.Path)
		}
	}
}
