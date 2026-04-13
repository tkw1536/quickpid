package apitest

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/server"
)

type harness struct {
	srv  *httptest.Server
	base string
	now  string
}

func newHarness(t *testing.T, factory ResolverFactory) *harness {
	t.Helper()
	srv := newServer(t, factory(t))
	return &harness{
		srv:  srv,
		base: srv.URL + MountPath,
		now:  "2020-01-02T03:04:05Z",
	}
}

func (h *harness) createNamespace(t *testing.T, name string) api.NamespaceResponse {
	t.Helper()
	body := mustMarshal(t, api.NamespaceCreateRequest{Name: name})
	resp := mustPOST(t, h.base+"/resolver/namespaces", body)
	defer resp.Body.Close()
	assertStatus(t, resp, http.StatusCreated)
	var got api.NamespaceResponse
	decodeJSON(t, resp.Body, &got)
	return got
}

func newServer(tb testing.TB, res api.Resolver) *httptest.Server {
	tb.Helper()
	opts := newServerOptions()
	opts.MountPath = MountPath
	mux := http.NewServeMux()
	apiHandler := server.NewHandler(opts, res)
	mux.Handle(MountPath+"/", http.StripPrefix(MountPath, apiHandler))
	mux.Handle("GET "+MountPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, MountPath+"/", http.StatusMovedPermanently)
	}))
	srv := httptest.NewServer(mux)
	tb.Cleanup(srv.Close)
	return srv
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
	if opts.GeneratePID == nil {
		opts.GeneratePID = newDeterministicPIDGen("pid")
	}
	if opts.Now == nil {
		fixed := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
		opts.Now = func() time.Time { return fixed }
	}
	return opts
}

// NewDeterministicPIDGen returns a deterministic PID generator suitable for tests.
// Each call returns prefix + a zero-padded counter starting at 1, e.g. "pid0001".
func newDeterministicPIDGen(prefix string) func() (string, error) {
	var n uint64
	return func() (string, error) {
		v := atomic.AddUint64(&n, 1)
		return fmt.Sprintf("%s%04d", prefix, v), nil
	}
}

func (h *harness) createResource(t *testing.T, namespace string, req api.ResourceCreateRequest) api.ResourceResponse {
	t.Helper()
	body := mustMarshal(t, req)
	u := fmt.Sprintf("%s/resolver/namespaces/%s/resources", h.base, namespace)
	resp := mustPOST(t, u, body)
	defer resp.Body.Close()
	assertStatus(t, resp, http.StatusCreated)
	var got api.ResourceResponse
	decodeJSON(t, resp.Body, &got)
	return got
}

func (h *harness) updateResource(t *testing.T, namespace, pid string, req api.ResourceUpdateRequest) api.ResourceResponse {
	t.Helper()
	body := mustMarshal(t, req)
	u := fmt.Sprintf("%s/resolver/namespaces/%s/resources/%s", h.base, namespace, pid)
	resp := mustPATCH(t, u, body)
	defer resp.Body.Close()
	assertStatus(t, resp, http.StatusOK)
	var got api.ResourceResponse
	decodeJSON(t, resp.Body, &got)
	return got
}
