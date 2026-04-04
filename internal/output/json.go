package output

import (
	"encoding/json"
	"io"

	"github.com/alecthomas/chroma/v2/quick"
)

// RenderJSON writes v as indented JSON to w. When isTTY is true,
// JSON output is syntax-highlighted for terminal display.
func RenderJSON(w io.Writer, v any, isTTY bool) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if isTTY {
		return quick.Highlight(w, string(data)+"\n", "json", "terminal256", "monokai")
	}
	_, err = w.Write(append(data, '\n'))
	return err
}
