package agent

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"
)

//go:embed guides
var guidesFS embed.FS

// GuideNames lists available guide names, populated at init time.
var GuideNames []string

func init() {
	entries, err := guidesFS.ReadDir("guides")
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		GuideNames = append(GuideNames, name)
	}
}

// Guide returns the content of a named guide. Returns an error listing
// available guides if the name is not found.
func Guide(name string) (string, error) {
	data, err := guidesFS.ReadFile("guides/" + name + ".md")
	if err != nil {
		return "", fmt.Errorf("unknown guide %q (available: %s)", name, strings.Join(GuideNames, ", "))
	}
	return string(data), nil
}
