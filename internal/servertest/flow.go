//spellchecker:words servertest
package servertest

//spellchecker:words slices testing time github bicpid backend authentication internal httpfixture server service
import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/tkw1536/bicpid/backend"
	"github.com/tkw1536/bicpid/backend/authentication"
	"github.com/tkw1536/bicpid/internal/httpfixture"
	"github.com/tkw1536/bicpid/pid"
	"github.com/tkw1536/bicpid/server"
	"github.com/tkw1536/bicpid/service"
)

// flow describes an HTTP test flow in terms.
type flow struct {

	// General metadata about the flow.
	Name    string `json:"name"`
	Comment string `json:"comment,omitempty"`

	// Steps are the HTTP tests to run against the server.
	Steps []struct {
		Name string `json:"name"`

		Config struct {
			NamespaceIDs []string  `json:"namespaceIDs,omitzero"`
			PIDs         []string  `json:"pids,omitzero"`
			Now          time.Time `json:"now,omitzero"`
			InfoEnabled  bool      `json:"infoEnabled"`
		} `json:"config"`

		Limits service.Limits `json:"limits,omitzero"`

		httpfixture.Fixture
	} `json:"steps"`
}

func (f flow) Run(t *testing.T, b backend.ResolverBackend) {
	t.Helper()

	var runtime testRuntime
	svc := service.New(b, authentication.NewInMemoryBackend(), &runtime, service.Options{})
	handler := server.NewServer(server.Options{}, svc, nil)

	for _, s := range f.Steps {

		// setup server options for this step
		handler.SetOptions(server.Options{
			Limits:      s.Limits,
			InfoEnabled: s.Config.InfoEnabled,
		})

		// update the runtime for this handler
		runtime.now = s.Config.Now
		runtime.namespaceIDs = slices.Clone(s.Config.NamespaceIDs)
		runtime.pids = slices.Clone(s.Config.PIDs)

		t.Run(s.Name, func(t *testing.T) {
			runtime.t = t

			s.Fixture.Run(t, handler)

			if !runtime.now.IsZero() && !runtime.usedNow {
				t.Errorf("now: %s was not used (this is an error in the testcases themselves)", runtime.now)
			}
			if runtime.namespaceIDs != nil && !runtime.usedNamespaceIDs {
				t.Errorf("namespaceIDs: %v was not used (this is an error in the testcases themselves)", runtime.namespaceIDs)
			}
			if runtime.pids != nil && !runtime.usedPIDs {
				t.Errorf("pids: %v was not used (this is an error in the testcases themselves)", runtime.pids)
			}

			runtime.t = nil
		})
	}
}

// testRuntime is a [service.Runtime] used during testing.
type testRuntime struct {
	t *testing.T

	now     time.Time
	usedNow bool

	namespaceIDs     []string
	usedNamespaceIDs bool

	pids     []string
	usedPIDs bool
}

func (r *testRuntime) NewNamespaceID() (string, error) {
	r.usedNamespaceIDs = true
	if r.namespaceIDs == nil {
		r.t.Fatalf("namespaceIDs: is not set (this is an error in the testcases themselves)")
	}
	if len(r.namespaceIDs) == 0 {
		return "", fmt.Errorf("no more namespace IDs configured")
	}
	id := r.namespaceIDs[0]
	r.namespaceIDs = r.namespaceIDs[1:]
	return id, nil
}

func (r *testRuntime) NewPID(format pid.Format) (string, error) {
	r.usedPIDs = true
	if r.pids == nil {
		r.t.Fatalf("pids: is not set (this is an error in the testcases themselves)")
	}
	if len(r.pids) == 0 {
		return "", fmt.Errorf("no more PIDs configured")
	}
	id := r.pids[0]
	r.pids = r.pids[1:]
	return id, nil
}

func (r *testRuntime) Now() time.Time {
	r.usedNow = true
	if r.now.IsZero() {
		r.t.Fatalf("now: is not set (this is an error in the testcases themselves)")
	}
	return r.now
}
