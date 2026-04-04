package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		opts     FormatOpts
		expected FormatType
	}{
		{
			name:     "agent mode forces agent JSON",
			opts:     FormatOpts{AgentMode: true, Format: "plain"},
			expected: FormatAgentJSON,
		},
		{
			name:     "agent mode overrides explicit format",
			opts:     FormatOpts{AgentMode: true, Format: "csv"},
			expected: FormatAgentJSON,
		},
		{
			name:     "explicit json format",
			opts:     FormatOpts{Format: "json"},
			expected: FormatJSON,
		},
		{
			name:     "explicit csv format",
			opts:     FormatOpts{Format: "csv"},
			expected: FormatCSV,
		},
		{
			name:     "default is plain",
			opts:     FormatOpts{Format: "plain"},
			expected: FormatPlain,
		},
		{
			name:     "empty format defaults to plain",
			opts:     FormatOpts{},
			expected: FormatPlain,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, DetectFormat(tt.opts))
		})
	}
}
