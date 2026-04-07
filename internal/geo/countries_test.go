package geo_test

import (
	"testing"

	"github.com/matcra587/peerscout/internal/geo"
	"github.com/stretchr/testify/assert"
)

func TestCountryName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{"united states", "US", "United States"},
		{"united kingdom", "GB", "United Kingdom"},
		{"germany", "DE", "Germany"},
		{"australia", "AU", "Australia"},
		{"japan", "JP", "Japan"},
		{"lowercase code", "us", "United States"},
		{"unknown code", "ZZ", ""},
		{"empty code", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, geo.CountryName(tt.code))
		})
	}
}
