package apitest

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tkw1536/quickpid/internal/steptest"
	"github.com/tkw1536/quickpid/pid"
	"github.com/tkw1536/quickpid/server"
)

type harness struct {
	srv     *httptest.Server
	base    string
	now     string
	backend ResolverFactory

	opts server.Options

	namespaceIDs []string
	pids         []string
}

func newHarness(t *testing.T, factory ResolverFactory) *harness {
	t.Helper()
	opts := newServerOptions()
	opts.MountPath = MountPath
	return &harness{
		now:     "2020-01-02T03:04:05Z",
		backend: factory,
		opts:    opts,
	}
}

func (h *harness) runSteps(t *testing.T, steps []steptest.Step) {
	t.Helper()
	h.ensureStarted(t)
	steptest.New(h.base, http.DefaultClient).Run(t, steps)
}

func (h *harness) ensureStarted(t *testing.T) {
	t.Helper()
	if h.srv != nil {
		return
	}
	if h.opts.NewNamespaceID == nil || h.opts.NewPID == nil {
		t.Fatalf("harness randomness not configured: call h.useFixedRandomness(...) at the start of the flow")
	}

	res := h.backend(t)
	mux := http.NewServeMux()
	apiHandler := server.NewHandler(h.opts, res)
	mux.Handle(MountPath+"/", http.StripPrefix(MountPath, apiHandler))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	h.srv = srv
	h.base = srv.URL + MountPath
}

// useFixedRandomness installs deterministic generators for namespace IDs and PIDs.
// Each call to NewNamespaceID / NewPID consumes one element from the respective slice.
func (h *harness) useFixedRandomness(t *testing.T, namespaceIDs []string, pids []string) {
	t.Helper()
	if h.srv != nil {
		t.Fatalf("cannot set randomness after server start")
	}
	h.namespaceIDs = append([]string(nil), namespaceIDs...)
	h.pids = append([]string(nil), pids...)

	h.opts.NewNamespaceID = func() (string, error) {
		if len(h.namespaceIDs) == 0 {
			return "", fmt.Errorf("no more namespace IDs configured")
		}
		id := h.namespaceIDs[0]
		h.namespaceIDs = h.namespaceIDs[1:]
		return id, nil
	}
	h.opts.NewPID = func(_ pid.Format) (string, error) {
		if len(h.pids) == 0 {
			return "", fmt.Errorf("no more PIDs configured")
		}
		id := h.pids[0]
		h.pids = h.pids[1:]
		return id, nil
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
