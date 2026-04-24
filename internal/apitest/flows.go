package apitest

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/tkw1536/quickpid/api"
)

func flowListNamespaces(t *testing.T, h *harness) {
	t.Helper()

	t.Run("empty", func(t *testing.T) {
		resp := mustGET(t, h.base+"/resolver/namespaces")
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)
		got := decodeJSON[api.PaginatedNamespacesResponse](t, resp.Body)
		want := api.PaginatedNamespacesResponse{Total: 0, Offset: 0, Items: []api.NamespaceResponse{}}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("namespaces: got %+v want %+v", got, want)
		}
	})

	h.createNamespace(t, "a")
	h.createNamespace(t, "b")
	h.createNamespace(t, "c")
	h.createNamespace(t, "d")
	h.createNamespace(t, "e")

	t.Run("pagination_defaultLimit", func(t *testing.T) {
		resp := mustGET(t, h.base+"/resolver/namespaces")
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)
		got := decodeJSON[api.PaginatedNamespacesResponse](t, resp.Body)
		want := api.PaginatedNamespacesResponse{
			Total:  5,
			Offset: 0,
			Items: []api.NamespaceResponse{
				{Name: "a", PIDFormat: api.PIDFormat{Pattern: "***-***", Characters: api.PIDCharactersFull}, DateCreated: h.now},
				{Name: "b", PIDFormat: api.PIDFormat{Pattern: "***-***", Characters: api.PIDCharactersFull}, DateCreated: h.now},
			},
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("namespaces: got %+v want %+v", got, want)
		}
	})

	t.Run("pagination_clampMaxLimit", func(t *testing.T) {
		resp := mustGET(t, h.base+"/resolver/namespaces?limit=999")
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)
		got := decodeJSON[api.PaginatedNamespacesResponse](t, resp.Body)
		want := api.PaginatedNamespacesResponse{
			Total:  5,
			Offset: 0,
			Items: []api.NamespaceResponse{
				{Name: "a", PIDFormat: api.PIDFormat{Pattern: "***-***", Characters: api.PIDCharactersFull}, DateCreated: h.now},
				{Name: "b", PIDFormat: api.PIDFormat{Pattern: "***-***", Characters: api.PIDCharactersFull}, DateCreated: h.now},
				{Name: "c", PIDFormat: api.PIDFormat{Pattern: "***-***", Characters: api.PIDCharactersFull}, DateCreated: h.now},
			},
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("namespaces: got %+v want %+v", got, want)
		}
	})

	t.Run("pagination_offsetPastEnd", func(t *testing.T) {
		resp := mustGET(t, h.base+"/resolver/namespaces?offset=5")
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)
		got := decodeJSON[api.PaginatedNamespacesResponse](t, resp.Body)
		want := api.PaginatedNamespacesResponse{Total: 5, Offset: 5, Items: []api.NamespaceResponse{}}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("namespaces: got %+v want %+v", got, want)
		}
	})

	t.Run("invalidQuery_limit", func(t *testing.T) {
		for _, q := range []string{"limit=0", "limit=-1", "limit=abc"} {
			resp := mustGET(t, h.base+"/resolver/namespaces?"+q)
			defer resp.Body.Close()
			assertStatus(t, resp, http.StatusBadRequest)
			assertErrorJSON(t, resp, "invalid query parameter \"limit\"")
		}
	})

	t.Run("invalidQuery_offset", func(t *testing.T) {
		for _, q := range []string{"offset=-1", "offset=abc"} {
			resp := mustGET(t, h.base+"/resolver/namespaces?"+q)
			defer resp.Body.Close()
			assertStatus(t, resp, http.StatusBadRequest)
			assertErrorJSON(t, resp, "invalid query parameter \"offset\"")
		}
	})
}

func flowCreateNamespace(t *testing.T, h *harness) {
	t.Helper()

	t.Run("success", func(t *testing.T) {
		got := h.createNamespace(t, "flow-ns")
		want := api.NamespaceResponse{Name: "flow-ns", PIDFormat: api.PIDFormat{Pattern: "***-***", Characters: api.PIDCharactersFull}, DateCreated: h.now}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("namespace: got %+v want %+v", got, want)
		}
	})

	t.Run("conflict", func(t *testing.T) {
		_ = h.createNamespace(t, "dup-ns")
		body := mustMarshal(t, api.NamespaceCreateRequest{
			Name: "dup-ns",
			PIDFormat: api.PIDFormat{
				Pattern:    "***-***",
				Characters: api.PIDCharactersFull,
			},
		})
		resp := mustPOST(t, h.base+"/resolver/namespaces", body)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusConflict)
		assertErrorJSON(t, resp, "namespace already exists")
	})

	t.Run("invalidNamespace", func(t *testing.T) {
		body := mustMarshal(t, api.NamespaceCreateRequest{Name: "bad/ns"})
		resp := mustPOST(t, h.base+"/resolver/namespaces", body)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusBadRequest)
		assertErrorJSON(t, resp, "invalid namespace")
	})

	t.Run("emptyBody", func(t *testing.T) {
		resp := mustPOST(t, h.base+"/resolver/namespaces", "")
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusBadRequest)
		assertErrorJSON(t, resp, "empty request body")
	})

	t.Run("trailingJSON", func(t *testing.T) {
		resp := mustPOST(t, h.base+"/resolver/namespaces", `{"name":"x","pid_format":{"pattern":"***","characters":"full"}} {"name":"y"}`)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusBadRequest)
		assertErrorJSON(t, resp, "trailing JSON")
	})

	t.Run("tooLargeBody", func(t *testing.T) {
		body := `{"name":"` + strings.Repeat("a", 512) + `"}`
		resp := mustPOST(t, h.base+"/resolver/namespaces", body)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusRequestEntityTooLarge)
		assertErrorJSON(t, resp, "request payload too large")
	})
}

func flowListResources(t *testing.T, h *harness) {
	t.Helper()

	ns := "list-res-ns"
	_ = h.createNamespace(t, ns)

	t.Run("namespaceMissing", func(t *testing.T) {
		resp := mustGET(t, h.base+"/resolver/namespaces/missing-ns/resources")
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusNotFound)
		assertErrorJSON(t, resp, "namespace not found")
	})

	t.Run("empty", func(t *testing.T) {
		u := fmt.Sprintf("%s/resolver/namespaces/%s/resources", h.base, ns)
		resp := mustGET(t, u)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)
		got := decodeJSON[api.PaginatedResourcesResponse](t, resp.Body)
		want := api.PaginatedResourcesResponse{Total: 0, Offset: 0, Items: []api.ResourceResponse{}}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("resources: got %+v want %+v", got, want)
		}
	})

	_ = h.createResource(t, ns, api.ResourceCreateRequest{
		URL:      "https://example.com/a",
		Metadata: new("ext-1@sys-a"),
		Tag:      "alpha",
	})
	_ = h.createResource(t, ns, api.ResourceCreateRequest{
		URL:      "https://example.com/b",
		Metadata: new("ext-2@sys-a"),
		Tag:      "beta",
	})
	_ = h.createResource(t, ns, api.ResourceCreateRequest{
		URL:      "https://example.com/empty-tag",
		Metadata: nil,
	})
	_ = h.createResource(t, ns, api.ResourceCreateRequest{
		URL:      "https://example.com/c",
		Metadata: new("ext-4@sys-a"),
		Tag:      "alpha",
	})
	_ = h.createResource(t, ns, api.ResourceCreateRequest{
		URL:      "https://example.com/d",
		Metadata: new("ext-5@sys-a"),
		Tag:      "alpha",
	})

	t.Run("pagination_defaultLimit", func(t *testing.T) {
		u := fmt.Sprintf("%s/resolver/namespaces/%s/resources", h.base, ns)
		resp := mustGET(t, u)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)
		got := decodeJSON[api.PaginatedResourcesResponse](t, resp.Body)
		want := api.PaginatedResourcesResponse{
			Total:  5,
			Offset: 0,
			Items: []api.ResourceResponse{
				{
					PID:         "59x-79p",
					URL:         "https://example.com/c",
					Metadata:    new("ext-4@sys-a"),
					DateCreated: h.now,
					DateUpdated: h.now,
					Tag:         "alpha",
					Deleted:     false,
				},
				{
					PID:         "651-pd3",
					URL:         "https://example.com/d",
					Metadata:    new("ext-5@sys-a"),
					DateCreated: h.now,
					DateUpdated: h.now,
					Tag:         "alpha",
					Deleted:     false,
				},
			},
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("resources: got %+v want %+v", got, want)
		}
	})

	t.Run("pagination_clampMaxLimit", func(t *testing.T) {
		u := fmt.Sprintf("%s/resolver/namespaces/%s/resources?limit=999", h.base, ns)
		resp := mustGET(t, u)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)
		got := decodeJSON[api.PaginatedResourcesResponse](t, resp.Body)
		if got.Total != 5 || got.Offset != 0 || len(got.Items) != 3 {
			t.Fatalf("resources: %+v", got)
		}
	})

	t.Run("pagination_offsetPastEnd", func(t *testing.T) {
		u := fmt.Sprintf("%s/resolver/namespaces/%s/resources?offset=5", h.base, ns)
		resp := mustGET(t, u)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)
		got := decodeJSON[api.PaginatedResourcesResponse](t, resp.Body)
		want := api.PaginatedResourcesResponse{Total: 5, Offset: 5, Items: []api.ResourceResponse{}}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("resources: got %+v want %+v", got, want)
		}
	})

	t.Run("tagOmitted", func(t *testing.T) {
		u := fmt.Sprintf("%s/resolver/namespaces/%s/resources", h.base, ns)
		resp := mustGET(t, u)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)
		got := decodeJSON[api.PaginatedResourcesResponse](t, resp.Body)
		if got.Total != 5 {
			t.Fatalf("resources: %+v", got)
		}
	})

	t.Run("tagFilter", func(t *testing.T) {
		u := fmt.Sprintf("%s/resolver/namespaces/%s/resources?tag=alpha&limit=999", h.base, ns)
		resp := mustGET(t, u)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)
		got := decodeJSON[api.PaginatedResourcesResponse](t, resp.Body)
		if got.Total != 3 || len(got.Items) != 3 {
			t.Fatalf("filtered: %+v", got)
		}
		for _, it := range got.Items {
			if it.Tag != "alpha" {
				t.Fatalf("want alpha tags, got %+v", got)
			}
		}
	})

	t.Run("tagNoMatch", func(t *testing.T) {
		u := fmt.Sprintf("%s/resolver/namespaces/%s/resources?tag=other", h.base, ns)
		resp := mustGET(t, u)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)
		got := decodeJSON[api.PaginatedResourcesResponse](t, resp.Body)
		want := api.PaginatedResourcesResponse{Total: 0, Offset: 0, Items: []api.ResourceResponse{}}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("filtered: got %+v want %+v", got, want)
		}
	})

	t.Run("tagEmptyMeansEmptyString", func(t *testing.T) {
		u := fmt.Sprintf("%s/resolver/namespaces/%s/resources?tag=&limit=999", h.base, ns)
		resp := mustGET(t, u)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)
		got := decodeJSON[api.PaginatedResourcesResponse](t, resp.Body)
		if got.Total != 1 || len(got.Items) != 1 || got.Items[0].Tag != "" || got.Items[0].URL != "https://example.com/empty-tag" {
			t.Fatalf("empty tag filter: %+v", got)
		}
	})
}

func flowCreateResource(t *testing.T, h *harness) {
	t.Helper()
	ns := "create-res-ns"
	_ = h.createNamespace(t, ns)

	t.Run("success", func(t *testing.T) {
		got := h.createResource(t, ns, api.ResourceCreateRequest{
			URL:      "https://example.com/a",
			Metadata: new("ext-1@sys-a"),
			Tag:      "alpha",
		})
		want := api.ResourceResponse{
			PID:         "yd6-lc0",
			URL:         "https://example.com/a",
			Metadata:    new("ext-1@sys-a"),
			DateCreated: h.now,
			DateUpdated: h.now,
			Tag:         "alpha",
			Deleted:     false,
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("resource: got %+v want %+v", got, want)
		}
	})

	t.Run("metadataNull_roundTrips", func(t *testing.T) {
		got := h.createResource(t, ns, api.ResourceCreateRequest{
			URL:      "https://example.com/metadata-null",
			Metadata: nil,
			Tag:      "alpha",
		})
		want := api.ResourceResponse{
			PID:         "tha-yrf",
			URL:         "https://example.com/metadata-null",
			Metadata:    nil,
			DateCreated: h.now,
			DateUpdated: h.now,
			Tag:         "alpha",
			Deleted:     false,
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("resource: got %+v want %+v", got, want)
		}
	})

	t.Run("hexPattern_isLowercaseHex", func(t *testing.T) {
		ns := "hex-res-ns"
		body := mustMarshal(t, api.NamespaceCreateRequest{
			Name: ns,
			PIDFormat: api.PIDFormat{
				Pattern:    "********-****-****-****-************",
				Characters: api.PIDCharactersHex,
			},
		})
		resp := mustPOST(t, h.base+"/resolver/namespaces", body)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusCreated)

		got := h.createResource(t, ns, api.ResourceCreateRequest{
			URL:      "https://example.com/hex",
			Metadata: nil,
			Tag:      "hex",
		})
		if len(got.PID) != 36 || got.PID[8] != '-' || got.PID[13] != '-' || got.PID[18] != '-' || got.PID[23] != '-' {
			t.Fatalf("pid shape: %q", got.PID)
		}
		for i := 0; i < len(got.PID); i++ {
			if got.PID[i] == '-' {
				continue
			}
			c := got.PID[i]
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Fatalf("pid not lowercase hex: %q", got.PID)
			}
		}
		if got.URL != "https://example.com/hex" || got.Metadata != nil || got.DateCreated != h.now || got.DateUpdated != h.now || got.Tag != "hex" || got.Deleted != false {
			t.Fatalf("resource: %+v", got)
		}
	})

	t.Run("readable6_isDeterministic", func(t *testing.T) {
		ns := "readable6-res-ns"
		body := mustMarshal(t, api.NamespaceCreateRequest{
			Name: ns,
			PIDFormat: api.PIDFormat{
				Pattern:    "***-***",
				Characters: api.PIDCharactersReadable,
			},
		})
		resp := mustPOST(t, h.base+"/resolver/namespaces", body)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusCreated)

		got := h.createResource(t, ns, api.ResourceCreateRequest{
			URL:      "https://example.com/readable6",
			Metadata: nil,
			Tag:      "r6",
		})
		want := api.ResourceResponse{
			PID:         "g54-mn5",
			URL:         "https://example.com/readable6",
			Metadata:    nil,
			DateCreated: h.now,
			DateUpdated: h.now,
			Tag:         "r6",
			Deleted:     false,
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("resource: got %+v want %+v", got, want)
		}
	})

	t.Run("readable9_isDeterministic", func(t *testing.T) {
		ns := "readable9-res-ns"
		body := mustMarshal(t, api.NamespaceCreateRequest{
			Name: ns,
			PIDFormat: api.PIDFormat{
				Pattern:    "***-***-***",
				Characters: api.PIDCharactersReadable,
			},
		})
		resp := mustPOST(t, h.base+"/resolver/namespaces", body)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusCreated)

		got := h.createResource(t, ns, api.ResourceCreateRequest{
			URL:      "https://example.com/readable9",
			Metadata: nil,
			Tag:      "r9",
		})
		want := api.ResourceResponse{
			PID:         "r56-wn7-068",
			URL:         "https://example.com/readable9",
			Metadata:    nil,
			DateCreated: h.now,
			DateUpdated: h.now,
			Tag:         "r9",
			Deleted:     false,
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("resource: got %+v want %+v", got, want)
		}
	})

	t.Run("random64_isDeterministic", func(t *testing.T) {
		ns := "random64-res-ns"
		body := mustMarshal(t, api.NamespaceCreateRequest{
			Name: ns,
			PIDFormat: api.PIDFormat{
				Pattern:    "****************************************************************",
				Characters: api.PIDCharactersFull,
			},
		})
		resp := mustPOST(t, h.base+"/resolver/namespaces", body)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusCreated)

		got := h.createResource(t, ns, api.ResourceCreateRequest{
			URL:      "https://example.com/random64",
			Metadata: nil,
			Tag:      "r64",
		})
		want := api.ResourceResponse{
			PID:         "c6xgmqk2joj8sz1wfu0vnc14cw8dg9k8otfkh04hahopg09vjsxwxyclw5ctqxn0",
			URL:         "https://example.com/random64",
			Metadata:    nil,
			DateCreated: h.now,
			DateUpdated: h.now,
			Tag:         "r64",
			Deleted:     false,
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("resource: got %+v want %+v", got, want)
		}
	})

	t.Run("namespaceMissing", func(t *testing.T) {
		body := mustMarshal(t, api.ResourceCreateRequest{URL: "https://x"})
		resp := mustPOST(t, h.base+"/resolver/namespaces/absent-ns/resources", body)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusNotFound)
		assertErrorJSON(t, resp, "namespace not found")
	})

	t.Run("invalidNamespace", func(t *testing.T) {
		body := mustMarshal(t, api.ResourceCreateRequest{URL: "https://x"})
		resp := mustPOST(t, h.base+"/resolver/namespaces/bad.ns/resources", body)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusBadRequest)
		assertErrorJSON(t, resp, "invalid namespace")
	})

	t.Run("tooLargeBody", func(t *testing.T) {
		oversize := `{"url":"https://` + strings.Repeat("a", 512) + `","metadata":"x","tag":"z"}`
		resp := mustPOST(t, fmt.Sprintf("%s/resolver/namespaces/%s/resources", h.base, ns), oversize)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusRequestEntityTooLarge)
		assertErrorJSON(t, resp, "request payload too large")
	})
}

func flowBatchCreateResources(t *testing.T, h *harness) {
	t.Helper()
	ns := "batch-res-ns"
	_ = h.createNamespace(t, ns)

	t.Run("success", func(t *testing.T) {
		batch := []api.ResourceCreateRequest{
			{URL: "https://b1", Metadata: new("b1@batch"), Tag: "batch"},
			{URL: "https://b2", Metadata: nil},
		}
		body := mustMarshal(t, batch)
		u := fmt.Sprintf("%s/resolver/namespaces/%s/resources:batch", h.base, ns)
		resp := mustPOST(t, u, body)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusCreated)
		got := []api.ResourceResponse{}
		got = decodeJSON[[]api.ResourceResponse](t, resp.Body)
		want := []api.ResourceResponse{
			{
				PID:         "yd6-lc0",
				URL:         "https://b1",
				Metadata:    new("b1@batch"),
				DateCreated: h.now,
				DateUpdated: h.now,
				Tag:         "batch",
				Deleted:     false,
			},
			{
				PID:         "tha-yrf",
				URL:         "https://b2",
				Metadata:    nil,
				DateCreated: h.now,
				DateUpdated: h.now,
				Tag:         "",
				Deleted:     false,
			},
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("batch: got %+v want %+v", got, want)
		}
	})

	t.Run("tooManyItems", func(t *testing.T) {
		batch := []api.ResourceCreateRequest{
			{URL: "https://b1", Metadata: new("b1@batch")},
			{URL: "https://b2", Metadata: new("b2@batch")},
			{URL: "https://b3", Metadata: new("b3@batch")},
		}
		body := mustMarshal(t, batch)
		u := fmt.Sprintf("%s/resolver/namespaces/%s/resources:batch", h.base, ns)
		resp := mustPOST(t, u, body)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusBadRequest)
		assertErrorJSON(t, resp, "too many items: 3 > 2")
	})

	t.Run("namespaceMissing", func(t *testing.T) {
		body := mustMarshal(t, []api.ResourceCreateRequest{
			{URL: "https://x"},
		})
		u := h.base + "/resolver/namespaces/missing/resources:batch"
		resp := mustPOST(t, u, body)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusNotFound)
		assertErrorJSON(t, resp, "namespace not found")
	})

	t.Run("tooLargeBody", func(t *testing.T) {
		u := fmt.Sprintf("%s/resolver/namespaces/%s/resources:batch", h.base, ns)
		oversize := `[{"url":"https://` + strings.Repeat("a", 512) + `","metadata":"x"}]`
		resp := mustPOST(t, u, oversize)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusRequestEntityTooLarge)
		assertErrorJSON(t, resp, "request payload too large")
	})
}

func flowGetResource(t *testing.T, h *harness) {
	t.Helper()
	ns := "get-res-ns"
	_ = h.createNamespace(t, ns)
	created := h.createResource(t, ns, api.ResourceCreateRequest{
		URL:      "https://example.com/a",
		Metadata: new("ext-1@sys-a"),
		Tag:      "alpha",
	})

	t.Run("success", func(t *testing.T) {
		u := fmt.Sprintf("%s/resolver/namespaces/%s/resources/%s", h.base, ns, created.PID)
		resp := mustGET(t, u)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)
		got := decodeJSON[api.ResourceResponse](t, resp.Body)
		want := api.ResourceResponse{
			PID:         created.PID,
			URL:         "https://example.com/a",
			Metadata:    new("ext-1@sys-a"),
			DateCreated: h.now,
			DateUpdated: h.now,
			Tag:         "alpha",
			Deleted:     false,
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("resource: got %+v want %+v", got, want)
		}
	})

	t.Run("resourceNotFound", func(t *testing.T) {
		u := fmt.Sprintf("%s/resolver/namespaces/%s/resources/99999", h.base, ns)
		resp := mustGET(t, u)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusNotFound)
		assertErrorJSON(t, resp, "resource not found")
	})

	t.Run("invalidNamespace", func(t *testing.T) {
		u := fmt.Sprintf("%s/resolver/namespaces/bad.ns/resources/%s", h.base, created.PID)
		resp := mustGET(t, u)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusBadRequest)
		assertErrorJSON(t, resp, "invalid namespace")
	})

	t.Run("invalidPID", func(t *testing.T) {
		u := fmt.Sprintf("%s/resolver/namespaces/%s/resources/bad.pid", h.base, ns)
		resp := mustGET(t, u)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusBadRequest)
		assertErrorJSON(t, resp, "invalid pid")
	})
}

func flowUpdateResource(t *testing.T, h *harness) {
	t.Helper()
	ns := "update-res-ns"
	_ = h.createNamespace(t, ns)
	created := h.createResource(t, ns, api.ResourceCreateRequest{
		URL:      "https://example.com/a",
		Metadata: new("ext-1@sys-a"),
		Tag:      "alpha",
	})

	t.Run("success", func(t *testing.T) {
		got := h.updateResource(t, ns, created.PID, api.ResourceUpdateRequest{
			URL:      "https://example.com/updated",
			Metadata: new("ext-1b@sys-b"),
			Tag:      "beta",
			Deleted:  false,
		})
		want := api.ResourceResponse{
			PID:         created.PID,
			URL:         "https://example.com/updated",
			Metadata:    new("ext-1b@sys-b"),
			DateCreated: h.now,
			DateUpdated: h.now,
			Tag:         "beta",
			Deleted:     false,
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("resource: got %+v want %+v", got, want)
		}
	})

	t.Run("setMetadataNull", func(t *testing.T) {
		got := h.updateResource(t, ns, created.PID, api.ResourceUpdateRequest{
			URL:      "https://example.com/updated-null",
			Metadata: nil,
			Tag:      "beta",
			Deleted:  false,
		})
		want := api.ResourceResponse{
			PID:         created.PID,
			URL:         "https://example.com/updated-null",
			Metadata:    nil,
			DateCreated: h.now,
			DateUpdated: h.now,
			Tag:         "beta",
			Deleted:     false,
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("resource: got %+v want %+v", got, want)
		}
	})

	t.Run("resourceNotFound", func(t *testing.T) {
		body := mustMarshal(t, api.ResourceUpdateRequest{
			URL:      "https://example.com/updated",
			Metadata: new("ext-1b@sys-b"),
			Tag:      "beta",
			Deleted:  false,
		})
		u := fmt.Sprintf("%s/resolver/namespaces/%s/resources/99999", h.base, ns)
		resp := mustPATCH(t, u, body)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusNotFound)
		assertErrorJSON(t, resp, "resource not found")
	})

	t.Run("invalidPID", func(t *testing.T) {
		body := mustMarshal(t, api.ResourceUpdateRequest{
			URL:      "https://example.com/updated",
			Metadata: new("ext-1b@sys-b"),
			Tag:      "beta",
			Deleted:  false,
		})
		u := fmt.Sprintf("%s/resolver/namespaces/%s/resources/bad.pid", h.base, ns)
		resp := mustPATCH(t, u, body)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusBadRequest)
		assertErrorJSON(t, resp, "invalid pid")
	})

	t.Run("tooLargeBody", func(t *testing.T) {
		u := fmt.Sprintf("%s/resolver/namespaces/%s/resources/%s", h.base, ns, created.PID)
		oversize := `{"url":"https://` + strings.Repeat("a", 512) + `","metadata":"x","tag":"z","deleted":false}`
		resp := mustPATCH(t, u, oversize)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusRequestEntityTooLarge)
		assertErrorJSON(t, resp, "request payload too large")
	})
}
