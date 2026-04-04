package output

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderAgentJSON_Envelope(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := RenderAgentJSON(&buf, "find", []string{"peer1"}, nil)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &m))
	assert.Equal(t, true, m["success"])
	assert.Equal(t, "find", m["command"])
	assert.NotNil(t, m["data"])
}

func TestRenderAgentJSON_Compact(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := RenderAgentJSON(&buf, "list", []string{"cosmos", "dydx"}, nil)
	require.NoError(t, err)

	// Agent JSON should be compact (no indentation).
	output := buf.String()
	assert.NotContains(t, output, "  ")
	assert.Equal(t, byte('\n'), output[len(output)-1])
}
