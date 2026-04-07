package geo

import (
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
)

// CountryName returns the English name for an ISO 3166-1 alpha-2
// country code. Returns empty string for unknown or invalid codes.
func CountryName(code string) string {
	r, err := language.ParseRegion(code)
	if err != nil {
		return ""
	}
	name := display.English.Regions().Name(r)
	if name == "Unknown Region" {
		return ""
	}
	return name
}
