//spellchecker:words pidtest
package pidtest

//spellchecker:words encoding json sort testing github quickpid
import (
	"encoding/json/v2"
	"fmt"
	"io/fs"
	"sort"
	"testing"

	"github.com/tkw1536/quickpid"
)

// RunFlowTests starts an httptest server and runs a sequential suite of
// subtests against resolver HTTP routes.
//
// Each sub-test is known as a "flow" test and corresponds to a single .json
// file from the specification.
//
// If the server behaves as expected, then the test succeeds.
// If the server behaves differently, the test fails and debug output from the
// logger is printed to test output.
func RunFlowTests(t *testing.T, factory StoreFactory) {
	t.Helper()

	flows, err := loadTestData()
	if err != nil {
		t.Fatal(err)
	}

	for _, flow := range flows {
		t.Run(flow.Name, func(t *testing.T) {
			t.Parallel()

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
		return nil, fmt.Errorf("failed to read test data directory: %w", err)
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
			return nil, fmt.Errorf("failed to read test data file %q: %w", name, err)
		}
		var f flow
		if err := json.Unmarshal(b, &f); err != nil {
			return nil, fmt.Errorf("failed to unmarshal test data file %q: %w", name, err)
		}
		out = append(out, f)
	}
	return out, nil
}
