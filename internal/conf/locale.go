// conf/locale.go contains all locales application supports

package conf

import (
	"fmt"
	"strings"
)

var Locales = map[string]string{
	"Afrikaans":            "labels_af.txt",
	"Brazilian Portuguese": "labels_pt-br.txt",
	"Catalan":              "labels_ca.txt",
	"Czech":                "labels_cs.txt",
	"Chinese":              "labels_zh.txt",
	"Croatian":             "labels_hr.txt",
	"Danish":               "labels_da.txt",
	"Dutch":                "labels_nl.txt",
	"English":              "labels_en.txt",
	"Estonian":             "labels_et.txt",
	"Finnish":              "labels_fi.txt",
	"French":               "labels_fr.txt",
	"German":               "labels_de.txt",
	"Greek":                "labels_el.txt",
	"Hungarian":            "labels_hu.txt",
	"Icelandic":            "labels_is.txt",
	"Indonesian":           "labels_id.txt",
	"Italian":              "labels_it.txt",
	"Japanese":             "labels_ja.txt",
	"Latvian":              "labels_lv.txt",
	"Lithuanian":           "labels_lt.txt",
	"Norwegian":            "labels_no.txt",
	"Polish":               "labels_pl.txt",
	"Portuguese":           "labels_pt.txt",
	"Russian":              "labels_ru.txt",
	"Slovak":               "labels_sk.txt",
	"Slovenian":            "labels_sl.txt",
	"Spanish":              "labels_es.txt",
	"Swedish":              "labels_sv.txt",
	"Thai":                 "labels_th.txt",
	"Ukrainian":            "labels_uk.txt",
}

var LocaleCodes = map[string]string{
	"af":    "Afrikaans",
	"pt-br": "Brazilian Portuguese",
	"ca":    "Catalan",
	"cs":    "Czech",
	"zh":    "Chinese",
	"hr":    "Croatian",
	"da":    "Danish",
	"nl":    "Dutch",
	"el":    "Greek",
	"en":    "English",
	"et":    "Estonian",
	"fi":    "Finnish",
	"fr":    "French",
	"de":    "German",
	"hu":    "Hungarian",
	"is":    "Icelandic",
	"id":    "Indonesian",
	"it":    "Italian",
	"ja":    "Japanese",
	"lv":    "Latvian",
	"lt":    "Lithuanian",
	"no":    "Norwegian",
	"pl":    "Polish",
	"pt":    "Portuguese",
	"ru":    "Russian",
	"sk":    "Slovak",
	"sl":    "Slovenian",
	"es":    "Spanish",
	"sv":    "Swedish",
	"th":    "Thai",
	"uk":    "Ukrainian",
}

// NormalizeLocale normalizes the input locale string and matches it to a known locale code or full name.
func NormalizeLocale(inputLocale string) (string, error) {
	inputLocale = strings.ToLower(inputLocale)

	if _, exists := Locales[LocaleCodes[inputLocale]]; exists {
		return inputLocale, nil
	}

	for code, fullName := range LocaleCodes {
		if strings.EqualFold(fullName, inputLocale) {
			return code, nil
		}
	}

	fullLocale, exists := LocaleCodes[inputLocale]
	if !exists {
		return "", fmt.Errorf("unsupported locale: %s", inputLocale)
	}

	if _, exists := Locales[fullLocale]; !exists {
		return "", fmt.Errorf("locale code supported but no label file found: %s", fullLocale)
	}

	return inputLocale, nil
}
