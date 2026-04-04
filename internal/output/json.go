package output

import (
	"encoding/json"
	"io"

	"github.com/matcra587/peerscout/internal/agent"
)

// RenderAgentJSON wraps data in an agent envelope and writes compact JSON to w.
func RenderAgentJSON(w io.Writer, command string, data any, hints []string) error {
	env := agent.Success(command, data, hints)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(env)
}
