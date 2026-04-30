package apitest

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/internal/steptest"
	"github.com/tkw1536/quickpid/pid"
	"github.com/tkw1536/quickpid/server"
)

// flow describes an HTTP test flow in terms of deterministic randomness and steps.
type flow struct {

	// General metadata about the flow.
	Name    string `json:"name"`
	Comment string `json:"comment,omitempty"`

	// Steps are the HTTP tests to run against the server.
	Steps []flowStep `json:"steps"`
}

type flowStep struct {
	NamespaceIDs []string      `json:"namespaceIDs"`
	PIDs         []string      `json:"pids"`
	Now          time.Time     `json:"now"`
	Step         steptest.Step `json:"step"`
}

func (f flow) Run(t *testing.T, b backend.Backend) {
	t.Helper()

	opts := newServerOptions()

	var namespaceIDs []string
	var pids []string
	var now time.Time

	opts.NewNamespaceID = func() (string, error) {
		if len(namespaceIDs) == 0 {
			return "", fmt.Errorf("no more namespace IDs configured")
		}
		id := namespaceIDs[0]
		namespaceIDs = namespaceIDs[1:]
		return id, nil
	}
	opts.NewPID = func(_ pid.Format) (string, error) {
		if len(pids) == 0 {
			return "", fmt.Errorf("no more PIDs configured")
		}
		id := pids[0]
		pids = pids[1:]
		return id, nil
	}
	opts.Now = func() time.Time {
		return now
	}

	handler := server.NewHandler(opts, b)
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	runner := steptest.New(srv.URL, http.DefaultClient)
	for _, s := range f.Steps {
		namespaceIDs = append([]string(nil), s.NamespaceIDs...)
		pids = append([]string(nil), s.PIDs...)
		now = s.Now
		runner.RunStep(t, s.Step)
	}
}

func newServerOptions() server.Options {
	var opts server.Options
	if opts.Limits.DefaultPageLimit == 0 {
		opts.Limits.DefaultPageLimit = 2
	}
	if opts.Limits.MaxPageLimit == 0 {
		opts.Limits.MaxPageLimit = 3
	}
	if opts.Limits.MaxBatchItems == 0 {
		opts.Limits.MaxBatchItems = 2
	}
	if opts.Limits.MaxBodyBytes == 0 {
		opts.Limits.MaxBodyBytes = 256
	}
	if opts.Now == nil {
		fixed := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
		opts.Now = func() time.Time { return fixed }
	}
	return opts
}
