package apitest

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
)

//go:embed flows/*.json
var embeddedFlowsFS embed.FS

// loadEmbeddedFlows loads all flows that have been embedded
func loadEmbeddedFlows() ([]flow, error) {
	entries, err := fs.ReadDir(embeddedFlowsFS, "flows")
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	out := make([]flow, 0, len(names))
	for _, name := range names {
		b, err := embeddedFlowsFS.ReadFile("flows/" + name)
		if err != nil {
			return nil, err
		}
		var f flow
		if err := json.Unmarshal(b, &f); err != nil {
			return nil, fmt.Errorf("unmarshal %s: %w", name, err)
		}
		if f.Name == "" {
			return nil, fmt.Errorf("%s: missing flow.name", name)
		}
		out = append(out, f)
	}
	return out, nil
}
