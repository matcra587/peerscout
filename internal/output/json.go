package output

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/matcra587/peerscout/internal/agent"
)

// RenderJSON writes v as indented JSON to w. When isTTY is true,
// JSON output is syntax-highlighted for terminal display.
func RenderJSON(w io.Writer, v any, isTTY bool) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return err
	}
	if isTTY {
		return quick.Highlight(w, buf.String(), "json", "terminal256", "monokai")
	}
	_, err := w.Write(buf.Bytes())
	return err
}

// RenderAgentJSON wraps data in an agent envelope and writes compact JSON to w.
func RenderAgentJSON(w io.Writer, command string, data any, hints []string) error {
	env := agent.Success(command, data, hints)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(env)
}
