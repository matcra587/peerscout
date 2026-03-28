package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// RenderJSON writes v as indented JSON to w.
func RenderJSON(w io.Writer, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}
