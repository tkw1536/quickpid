// Package servertest runs tests for a specific [backend.Backend] implementation against the [server.Handler].
package servertest

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"testing"

	"github.com/tkw1536/quickpid/backend"
)

// BackendFactory is a function that creates a concrete new Resolver.
// It should call t.Fatal the test if it cannot create a resolver.
// It should use t.Cleanup if cleanup is needed after the test.
type BackendFactory = func(t *testing.T) backend.Backend

// TestBackend starts an httptest server for res and runs a sequential suite of
// subtests against resolver HTTP routes (namespaces, resources, batch, errors).
// Resolver implementations can call this from their tests with their concrete implementation.
func TestBackend(t *testing.T, factory BackendFactory) {
	t.Helper()

	flows, err := loadTestData()
	if err != nil {
		t.Fatal(err)
	}

	for _, flow := range flows {
		t.Run(flow.Name, func(t *testing.T) {
			b := factory(t)
			flow.Run(t, b)
		})
	}
}

//go:embed testdata/*.json
var embeddedFlowsFS embed.FS

// loadTestData loads all flows from the 'testdata' directory.
// These have been embedded in the binary.
func loadTestData() ([]flow, error) {
	entries, err := fs.ReadDir(embeddedFlowsFS, "testdata")
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
		b, err := embeddedFlowsFS.ReadFile("testdata/" + name)
		if err != nil {
			return nil, err
		}
		var f flow
		if err := json.Unmarshal(b, &f); err != nil {
			return nil, fmt.Errorf("unmarshal %s: %w", name, err)
		}
		out = append(out, f)
	}
	return out, nil
}
