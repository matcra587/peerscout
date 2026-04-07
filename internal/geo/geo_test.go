package geo_test

import (
	"testing"

	"github.com/matcra587/peerscout/internal/geo"
	"github.com/stretchr/testify/assert"
)

func TestExtractIPs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		peers    []string
		expected []string
	}{
		{
			name:     "standard peers",
			peers:    []string{"abc@1.2.3.4:26656", "def@5.6.7.8:26656"},
			expected: []string{"1.2.3.4", "5.6.7.8"},
		},
		{
			name:     "deduplicates shared IPs",
			peers:    []string{"abc@1.2.3.4:26656", "def@1.2.3.4:26656"},
			expected: []string{"1.2.3.4"},
		},
		{
			name:     "ipv6 bracket notation",
			peers:    []string{"abc@[::1]:26656"},
			expected: []string{"::1"},
		},
		{
			name:     "skips malformed peers",
			peers:    []string{"nope", "abc@1.2.3.4:26656"},
			expected: []string{"1.2.3.4"},
		},
		{
			name:     "empty input",
			peers:    []string{},
			expected: nil,
		},
		{
			name:     "nil input",
			peers:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, geo.ExtractIPs(tt.peers))
		})
	}
}
