package output

import (
	"encoding/json"
	"io"

	"github.com/matcra587/peerscout/internal/agent"
)

// PlainFunc renders data in a command-specific human-readable format.
type PlainFunc func(w io.Writer) error

// RenderOpts carries everything needed to render command output.
type RenderOpts struct {
	Command   string
	Data      any
	Hints     []string
	Format    FormatType
	PlainFunc PlainFunc
}

// Render writes command output in the format specified by opts.Format.
// Agent JSON and JSON are handled centrally. Plain and CSV delegate to
// opts.PlainFunc. When PlainFunc is nil, plain falls back to JSON.
func Render(w io.Writer, opts RenderOpts) error {
	switch opts.Format {
	case FormatAgentJSON:
		env := agent.Success(opts.Command, opts.Data, opts.Hints)
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		return enc.Encode(env)

	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.SetEscapeHTML(false)
		return enc.Encode(opts.Data)

	default:
		if opts.PlainFunc != nil {
			return opts.PlainFunc(w)
		}
		return json.NewEncoder(w).Encode(opts.Data)
	}
}
