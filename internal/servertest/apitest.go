// Package servertest runs tests for a specific [backend.ResolverBackend] implementation against the [server.Server].
//
//spellchecker:words servertest
package servertest

//spellchecker:words encoding json sort testing github bicpid quickpid backend
import (
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"testing"

	quickpid "github.com/tkw1536/bicpid"
	"github.com/tkw1536/bicpid/backend"
)

// StoreFactory creates a concrete [backend.Store] for HTTP integration tests.
type StoreFactory = func(t *testing.T) backend.Store

// TestBackend starts an httptest server and runs a sequential suite of
// subtests against resolver HTTP routes (namespaces, resources, batch, errors).
func TestBackend(t *testing.T, factory StoreFactory) {
	t.Helper()

	flows, err := loadTestData()
	if err != nil {
		t.Fatal(err)
	}

	for _, flow := range flows {
		t.Run(flow.Name, func(t *testing.T) {
			s := factory(t)
			flow.Run(t, s)
		})
	}
}

// loadTestData loads all flows from the 'testdata' directory.
// These have been embedded in the binary.
func loadTestData() ([]flow, error) {
	testDataFS := quickpid.GetTestData()

	entries, err := fs.ReadDir(testDataFS, ".")
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
		b, err := fs.ReadFile(testDataFS, name)
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
