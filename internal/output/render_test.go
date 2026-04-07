package output_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/matcra587/peerscout/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRender_AgentJSON(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	data := map[string]string{"key": "value"}

	err := output.Render(&buf, output.RenderOpts{
		Command:   "test",
		Data:      data,
		Format:    output.FormatAgentJSON,
		PlainFunc: func(w io.Writer) error { return errors.New("should not be called") },
	})
	require.NoError(t, err)

	var env map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &env))
	assert.Equal(t, true, env["success"])
	assert.Equal(t, "test", env["command"])
}

func TestRender_JSON(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	data := map[string]string{"key": "value"}

	err := output.Render(&buf, output.RenderOpts{
		Command: "test",
		Data:    data,
		Format:  output.FormatJSON,
	})
	require.NoError(t, err)

	var got map[string]string
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, "value", got["key"])
}

func TestRender_Plain(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	called := false

	err := output.Render(&buf, output.RenderOpts{
		Command: "test",
		Data:    "ignored",
		Format:  output.FormatPlain,
		PlainFunc: func(w io.Writer) error {
			called = true
			fmt.Fprintln(w, "hello")
			return nil
		},
	})
	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, "hello\n", buf.String())
}

func TestRender_PlainNilFunc(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	err := output.Render(&buf, output.RenderOpts{
		Command: "test",
		Data:    "fallback",
		Format:  output.FormatPlain,
	})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "fallback")
}

func TestRender_AgentJSON_NoHTMLEscape(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	data := map[string]string{"url": "https://example.com?a=1&b=2"}

	err := output.Render(&buf, output.RenderOpts{
		Command: "test",
		Data:    data,
		Format:  output.FormatAgentJSON,
	})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "&")
	assert.NotContains(t, buf.String(), `\u0026`)
}

func TestRender_Hints(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	err := output.Render(&buf, output.RenderOpts{
		Command: "test",
		Data:    "ok",
		Hints:   []string{"try this"},
		Format:  output.FormatAgentJSON,
	})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "try this")
}
