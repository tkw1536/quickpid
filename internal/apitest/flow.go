package apitest

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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
		} `json:"config"`

		Limits server.Limits `json:"limits"`

		httptester.TestCase
	} `json:"steps"`
}

func (f flow) Run(t *testing.T, b backend.Backend) {
	t.Helper()

	var opts server.Options
	var runtime testRuntime

	handler := server.NewHandler(opts, &runtime, b)

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	runner := httptester.New(srv.URL, http.DefaultClient)
	for _, s := range f.Steps {
		handler.SetOptions(server.Options{Limits: s.Limits})

		// setup the runtime for this step
		runtime.now = s.Config.Now
		runtime.namespaceIDs = append([]string(nil), s.Config.NamespaceIDs...)
		runtime.pids = append([]string(nil), s.Config.PIDs...)

		runner.Run(t, s.TestCase)
	}
}

// testRuntime is a [server.Runtime] used during testing.
type testRuntime struct {
	now          time.Time
	namespaceIDs []string
	pids         []string
}

func (r *testRuntime) NewNamespaceID() (string, error) {
	if len(r.namespaceIDs) == 0 {
		return "", fmt.Errorf("no more namespace IDs configured")
	}
	id := r.namespaceIDs[0]
	r.namespaceIDs = r.namespaceIDs[1:]
	return id, nil
}

func (r *testRuntime) NewPID(format pid.Format) (string, error) {
	if len(r.pids) == 0 {
		return "", fmt.Errorf("no more PIDs configured")
	}
	id := r.pids[0]
	r.pids = r.pids[1:]
	return id, nil
}

func (r *testRuntime) Now() time.Time {
	return r.now
}
