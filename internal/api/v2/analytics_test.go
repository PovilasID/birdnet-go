// analytics_test.go: Package api provides tests for API v2 analytics endpoints.

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"errors"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/tphakala/birdnet-go/internal/datastore"
)

// TestGetSpeciesSummary tests the species summary endpoint
func TestGetSpeciesSummary(t *testing.T) {
	// Setup
	e, mockDS, controller := setupTestEnvironment(t)

	// Create mock data
	firstSeen := time.Now().AddDate(0, -1, 0)
	lastSeen := time.Now().AddDate(0, 0, -1)

	mockSummaryData := []datastore.SpeciesSummaryData{
		{
			ScientificName: "Turdus migratorius",
			CommonName:     "American Robin",
			Count:          42,
			FirstSeen:      firstSeen,
			LastSeen:       lastSeen,
			AvgConfidence:  0.75,
			MaxConfidence:  0.85,
		},
		{
			ScientificName: "Cyanocitta cristata",
			CommonName:     "Blue Jay",
			Count:          27,
			FirstSeen:      time.Now().AddDate(0, -2, 0),
			LastSeen:       time.Now(),
			AvgConfidence:  0.82,
			MaxConfidence:  0.92,
		},
	}

	// Setup mock expectations
	mockDS.On("GetSpeciesSummaryData").Return(mockSummaryData, nil)

	// Create a request
	req := httptest.NewRequest(http.MethodGet, "/api/v2/analytics/species", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// We need to bypass auth middleware for this test
	handler := func(c echo.Context) error {
		return controller.GetSpeciesSummary(c)
	}

	// Test
	if assert.NoError(t, handler(c)) {
		// Check response
		assert.Equal(t, http.StatusOK, rec.Code)

		// Parse response body
		var response []map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)

		// Check response content
		assert.Len(t, response, 2)
		assert.Equal(t, "Turdus migratorius", response[0]["scientific_name"])
		assert.Equal(t, "American Robin", response[0]["common_name"])
		assert.Equal(t, float64(42), response[0]["count"])
		assert.Equal(t, "Cyanocitta cristata", response[1]["scientific_name"])
		assert.Equal(t, "Blue Jay", response[1]["common_name"])
		assert.Equal(t, float64(27), response[1]["count"])
	}

	// Verify mock expectations
	mockDS.AssertExpectations(t)
}

// TestGetHourlyAnalytics tests the hourly analytics endpoint
func TestGetHourlyAnalytics(t *testing.T) {
	// Setup
	e, mockDS, controller := setupTestEnvironment(t)

	// Create mock data
	date := "2023-01-01"
	species := "Turdus migratorius"

	mockHourlyData := []datastore.HourlyAnalyticsData{
		{
			Hour:  0,
			Count: 5,
		},
		{
			Hour:  1,
			Count: 3,
		},
	}

	// Setup mock expectations
	mockDS.On("GetHourlyAnalyticsData", date, species).Return(mockHourlyData, nil)

	// Create a request
	req := httptest.NewRequest(http.MethodGet, "/api/v2/analytics/hourly?date=2023-01-01&species=Turdus+migratorius", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v2/analytics/hourly")
	c.QueryParams().Set("date", date)
	c.QueryParams().Set("species", species)

	// We need to bypass auth middleware for this test
	handler := func(c echo.Context) error {
		return controller.GetHourlyAnalytics(c)
	}

	// Test
	if assert.NoError(t, handler(c)) {
		// Check response
		assert.Equal(t, http.StatusOK, rec.Code)

		// Parse response body - the actual implementation returns a single object, not an array
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)

		// Check response content
		assert.Equal(t, date, response["date"])
		assert.Equal(t, species, response["species"])

		// Check the counts array
		counts, ok := response["counts"].([]interface{})
		assert.True(t, ok, "Expected counts to be an array")
		assert.Len(t, counts, 24, "Expected 24 hours in counts array")

		// Check specific hour counts that were set in our mock
		assert.Equal(t, float64(5), counts[0], "Hour 0 should have 5 counts")
		assert.Equal(t, float64(3), counts[1], "Hour 1 should have 3 counts")

		// Check the total
		assert.Equal(t, float64(8), response["total"], "Total should be sum of all counts")
	}

	// Verify mock expectations
	mockDS.AssertExpectations(t)
}

// TestGetDailyAnalytics tests the daily analytics endpoint
func TestGetDailyAnalytics(t *testing.T) {
	// Setup
	e, mockDS, controller := setupTestEnvironment(t)

	// Create mock data
	startDate := "2023-01-01"
	endDate := "2023-01-07"
	species := "Turdus migratorius"

	mockDailyData := []datastore.DailyAnalyticsData{
		{
			Date:  "2023-01-01",
			Count: 12,
		},
		{
			Date:  "2023-01-02",
			Count: 8,
		},
	}

	// Setup mock expectations
	mockDS.On("GetDailyAnalyticsData", startDate, endDate, species).Return(mockDailyData, nil)

	// Create a request
	req := httptest.NewRequest(http.MethodGet,
		"/api/v2/analytics/daily?start_date=2023-01-01&end_date=2023-01-07&species=Turdus+migratorius", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v2/analytics/daily")
	c.QueryParams().Set("start_date", startDate)
	c.QueryParams().Set("end_date", endDate)
	c.QueryParams().Set("species", species)

	// We need to bypass auth middleware for this test
	handler := func(c echo.Context) error {
		return controller.GetDailyAnalytics(c)
	}

	// Test
	if assert.NoError(t, handler(c)) {
		// Check response
		assert.Equal(t, http.StatusOK, rec.Code)

		// Parse response body - the actual implementation returns an object with a 'data' array
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)

		// Check response metadata
		assert.Equal(t, startDate, response["start_date"])
		assert.Equal(t, endDate, response["end_date"])
		assert.Equal(t, species, response["species"])
		assert.Equal(t, float64(20), response["total"]) // 12 + 8 = 20

		// Check data array
		data, ok := response["data"].([]interface{})
		assert.True(t, ok, "Expected data to be an array")
		assert.Len(t, data, 2, "Expected 2 items in data array")

		// Check first data item
		item1 := data[0].(map[string]interface{})
		assert.Equal(t, "2023-01-01", item1["date"])
		assert.Equal(t, float64(12), item1["count"])

		// Check second data item
		item2 := data[1].(map[string]interface{})
		assert.Equal(t, "2023-01-02", item2["date"])
		assert.Equal(t, float64(8), item2["count"])
	}

	// Verify mock expectations
	mockDS.AssertExpectations(t)
}

// TestGetDailyAnalyticsWithoutSpecies tests the daily analytics endpoint when no species is provided
// This tests the aggregated data behavior, which represents detection trends across all species
func TestGetDailyAnalyticsWithoutSpecies(t *testing.T) {
	// Setup
	e, mockDS, controller := setupTestEnvironment(t)

	// Create mock data
	startDate := "2023-01-01"
	endDate := "2023-01-07"

	mockDailyData := []datastore.DailyAnalyticsData{
		{
			Date:  "2023-01-07",
			Count: 45,
		},
		{
			Date:  "2023-01-06",
			Count: 38,
		},
		{
			Date:  "2023-01-05",
			Count: 42,
		},
	}

	// Setup mock expectations
	mockDS.On("GetDailyAnalyticsData", startDate, endDate, "").Return(mockDailyData, nil)

	// Create a request
	req := httptest.NewRequest(http.MethodGet,
		"/api/v2/analytics/daily?start_date=2023-01-01&end_date=2023-01-07", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v2/analytics/daily")
	c.QueryParams().Set("start_date", startDate)
	c.QueryParams().Set("end_date", endDate)

	// We need to bypass auth middleware for this test
	handler := func(c echo.Context) error {
		return controller.GetDailyAnalytics(c)
	}

	// Test
	if assert.NoError(t, handler(c)) {
		// Check response
		assert.Equal(t, http.StatusOK, rec.Code)

		// Parse response body
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)

		// Check response content
		data, ok := response["data"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, data, 3)
	}

	// Verify mock expectations
	mockDS.AssertExpectations(t)
}

// TestGetInvalidAnalyticsRequests tests analytics endpoints with invalid parameters
func TestGetInvalidAnalyticsRequests(t *testing.T) {
	// Setup
	e, _, controller := setupTestEnvironment(t)

	// Test cases
	testCases := []struct {
		name        string
		endpoint    string
		handler     func(echo.Context) error
		queryParams map[string]string
		expectCode  int
	}{
		{
			name:     "Missing date for hourly analytics",
			endpoint: "/api/v2/analytics/hourly",
			handler:  controller.GetHourlyAnalytics,
			queryParams: map[string]string{
				"species": "Turdus migratorius",
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name:     "Missing species for hourly analytics",
			endpoint: "/api/v2/analytics/hourly",
			handler:  controller.GetHourlyAnalytics,
			queryParams: map[string]string{
				"date": "2023-01-01",
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name:     "Invalid date format for hourly analytics",
			endpoint: "/api/v2/analytics/hourly",
			handler:  controller.GetHourlyAnalytics,
			queryParams: map[string]string{
				"date":    "01-01-2023", // Wrong format
				"species": "Turdus migratorius",
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name:     "Missing start_date for daily analytics",
			endpoint: "/api/v2/analytics/daily",
			handler:  controller.GetDailyAnalytics,
			queryParams: map[string]string{
				"end_date": "2023-01-07",
				"species":  "Turdus migratorius",
			},
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a request
			req := httptest.NewRequest(http.MethodGet, tc.endpoint, http.NoBody)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath(tc.endpoint)

			// Set query parameters
			for key, value := range tc.queryParams {
				c.QueryParams().Set(key, value)
			}

			// Call handler
			err := tc.handler(c)

			// Check if error handling works as expected
			var httpErr *echo.HTTPError
			if errors.As(err, &httpErr) {
				assert.Equal(t, tc.expectCode, httpErr.Code)
			} else {
				assert.Equal(t, tc.expectCode, rec.Code)
			}
		})
	}
}
