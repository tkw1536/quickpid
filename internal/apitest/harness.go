package apitest

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/server"
)

type harness struct {
	srv  *httptest.Server
	base string
	now  string
	rand io.Reader
}

func newHarness(t *testing.T, factory ResolverFactory) *harness {
	t.Helper()
	res := factory(t)
	opts, r := newServerOptions()
	opts.MountPath = MountPath

	mux := http.NewServeMux()
	apiHandler := server.NewHandler(opts, res)
	mux.Handle(MountPath+"/", http.StripPrefix(MountPath, apiHandler))
	mux.Handle("GET "+MountPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, MountPath+"/", http.StatusMovedPermanently)
	}))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &harness{
		srv:  srv,
		base: srv.URL + MountPath,
		now:  "2020-01-02T03:04:05Z",
		rand: r,
	}
}

func (h *harness) createNamespace(t *testing.T, name string) api.NamespaceResponse {
	t.Helper()
	body := mustMarshal(t, api.NamespaceCreateRequest{Name: name, PIDGenerator: api.PIDGeneratorLegacy})
	resp := mustPOST(t, h.base+"/resolver/namespaces", body)
	defer resp.Body.Close()
	assertStatus(t, resp, http.StatusCreated)
	return decodeJSON[api.NamespaceResponse](t, resp.Body)
}

func newServerOptions() (server.Options, io.Reader) {
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
	r := NewFakeRandReader()
	opts.Rand = r
	if opts.Now == nil {
		fixed := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
		opts.Now = func() time.Time { return fixed }
	}
	return opts, r
}

func (h *harness) createResource(t *testing.T, namespace string, req api.ResourceCreateRequest) api.ResourceResponse {
	t.Helper()
	body := mustMarshal(t, req)
	u := fmt.Sprintf("%s/resolver/namespaces/%s/resources", h.base, namespace)
	resp := mustPOST(t, u, body)
	defer resp.Body.Close()
	assertStatus(t, resp, http.StatusCreated)
	return decodeJSON[api.ResourceResponse](t, resp.Body)
}

func (h *harness) updateResource(t *testing.T, namespace, pid string, req api.ResourceUpdateRequest) api.ResourceResponse {
	t.Helper()
	body := mustMarshal(t, req)
	u := fmt.Sprintf("%s/resolver/namespaces/%s/resources/%s", h.base, namespace, pid)
	resp := mustPATCH(t, u, body)
	defer resp.Body.Close()
	assertStatus(t, resp, http.StatusOK)
	return decodeJSON[api.ResourceResponse](t, resp.Body)
}
