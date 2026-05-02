package servertest

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/internal/httptester"
	"github.com/tkw1536/quickpid/pid"
	"github.com/tkw1536/quickpid/server"
)

// flow describes an HTTP test flow in terms of deterministic randomness and steps.
type flow struct {

	// General metadata about the flow.
	Name    string `json:"name"`
	Comment string `json:"comment,omitempty"`

	// Steps are the HTTP tests to run against the server.
	Steps []struct {
		Name string `json:"name"`

		Config struct {
			NamespaceIDs []string  `json:"namespaceIDs"`
			PIDs         []string  `json:"pids"`
			Now          time.Time `json:"now"`
			// InfoEnabled mirrors [server.Options.InfoEnabled]: when true, GET /resolver returns [spec.InfoUnavailable].
			InfoEnabled bool `json:"infoEnabled"`
		} `json:"config"`

		Limits server.Limits `json:"limits"`

		httptester.TestCase
	} `json:"steps"`
}

func (f flow) Run(t *testing.T, b backend.Backend) {
	t.Helper()

	var (
		opts    server.Options
		runtime testRuntime
	)
	handler := server.NewHandler(opts, &runtime, b)

	for _, s := range f.Steps {
		// update the options for the handler
		handler.SetOptions(server.Options{
			Limits:      s.Limits,
			InfoEnabled: s.Config.InfoEnabled,
		})

		// update the runtime for this handler
		runtime.now = s.Config.Now
		runtime.namespaceIDs = slices.Clone(s.Config.NamespaceIDs)
		runtime.pids = slices.Clone(s.Config.PIDs)

		t.Run(s.Name, func(t *testing.T) {
			s.TestCase.Run(t, handler)

			if !runtime.now.IsZero() && !runtime.usedNow {
				t.Errorf("now: %s was not used", runtime.now)
			}
			if runtime.namespaceIDs != nil && !runtime.usedNamespaceIDs {
				t.Errorf("namespaceIDs: %v was not used", runtime.namespaceIDs)
			}
			if runtime.pids != nil && !runtime.usedPIDs {
				t.Errorf("pids: %v was not used", runtime.pids)
			}
		})
	}
}

// testRuntime is a [server.Runtime] used during testing.
type testRuntime struct {
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
		panic("namespaceIDs: is not set")
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
		panic("pids: is not set")
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
		panic("now: is not set")
	}
	return r.now
}
