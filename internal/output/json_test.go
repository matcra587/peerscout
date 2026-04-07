package output_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/matcra587/peerscout/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRender_AgentJSON_Envelope(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	data := map[string]string{"network": "cosmos"}
	err := output.Render(&buf, output.RenderOpts{
		Command: "find",
		Data:    data,
		Hints:   []string{"check list"},
		Format:  output.FormatAgentJSON,
	})
	require.NoError(t, err)

	var env map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &env))
	assert.Equal(t, true, env["success"])
	assert.Equal(t, "find", env["command"])
}

func TestRender_AgentJSON_Compact(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := output.Render(&buf, output.RenderOpts{
		Command: "test",
		Data:    map[string]int{"a": 1},
		Format:  output.FormatAgentJSON,
	})
	require.NoError(t, err)

	// json.Encoder.Encode appends a single newline — no pretty-printing.
	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	assert.Len(t, lines, 1)
}
