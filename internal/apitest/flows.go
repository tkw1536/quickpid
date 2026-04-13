package apitest

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tkw1536/quickpid/api"
)

// resolverFlow exercises every resolver HTTP route in order against srv, using dummy data.
func resolverFlow(t *testing.T, srv *httptest.Server) {
	t.Helper()
	base := srv.URL + MountPath
	ns := "flow-ns"
	pid1 := ""

	t.Run("listNamespaces_empty", func(t *testing.T) {
		resp := mustGET(t, base+"/resolver/namespaces")
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)
		var got api.PaginatedNamespacesResponse
		decodeJSON(t, resp.Body, &got)
		if got.Total != 0 || got.Offset != 0 || len(got.Items) != 0 {
			t.Fatalf("namespaces: %+v", got)
		}
	})

	t.Run("createNamespace", func(t *testing.T) {
		body := mustMarshal(t, api.NamespaceCreateRequest{Name: ns})
		resp := mustPOST(t, base+"/resolver/namespaces", body)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusCreated)
		var got api.NamespaceResponse
		decodeJSON(t, resp.Body, &got)
		if got.Name != ns || got.DateCreated == "" {
			t.Fatalf("namespace: %+v", got)
		}
	})

	t.Run("createNamespace_conflict", func(t *testing.T) {
		body := mustMarshal(t, api.NamespaceCreateRequest{Name: ns})
		resp := mustPOST(t, base+"/resolver/namespaces", body)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusConflict)
		assertErrorJSON(t, resp, "namespace already exists")
	})

	t.Run("listNamespaces_one", func(t *testing.T) {
		resp := mustGET(t, base+"/resolver/namespaces")
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)
		var got api.PaginatedNamespacesResponse
		decodeJSON(t, resp.Body, &got)
		if got.Total != 1 || got.Offset != 0 || len(got.Items) != 1 || got.Items[0].Name != ns {
			t.Fatalf("namespaces: %+v", got)
		}
	})

	t.Run("listResources_namespaceMissing", func(t *testing.T) {
		resp := mustGET(t, base+"/resolver/namespaces/missing-ns/resources")
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusNotFound)
		assertErrorJSON(t, resp, "namespace not found")
	})

	t.Run("listResources_empty", func(t *testing.T) {
		resp := mustGET(t, fmt.Sprintf("%s/resolver/namespaces/%s/resources", base, ns))
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)
		var got api.PaginatedResourcesResponse
		decodeJSON(t, resp.Body, &got)
		if got.Total != 0 || got.Offset != 0 || len(got.Items) != 0 {
			t.Fatalf("resources: %+v", got)
		}
	})

	t.Run("createResource", func(t *testing.T) {
		reqBody := mustMarshal(t, api.ResourceCreateRequest{
			URL:          "https://example.com/a",
			IdInTarget:   "ext-1",
			TargetSystem: "sys-a",
			Tag:          "alpha",
		})
		u := fmt.Sprintf("%s/resolver/namespaces/%s/resources", base, ns)
		resp := mustPOST(t, u, reqBody)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusCreated)
		var got api.ResourceResponse
		decodeJSON(t, resp.Body, &got)
		if got.PID == "" || got.URL != "https://example.com/a" || got.IdInTarget != "ext-1" ||
			got.TargetSystem != "sys-a" || got.Tag != "alpha" || got.Deleted {
			t.Fatalf("resource: %+v", got)
		}
		pid1 = got.PID
	})

	t.Run("createResource_namespaceMissing", func(t *testing.T) {
		reqBody := mustMarshal(t, api.ResourceCreateRequest{
			URL: "https://x", IdInTarget: "x", TargetSystem: "y",
		})
		u := base + "/resolver/namespaces/absent-ns/resources"
		resp := mustPOST(t, u, reqBody)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusNotFound)
		assertErrorJSON(t, resp, "namespace not found")
	})

	t.Run("listResources_one", func(t *testing.T) {
		resp := mustGET(t, fmt.Sprintf("%s/resolver/namespaces/%s/resources", base, ns))
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)
		var got api.PaginatedResourcesResponse
		decodeJSON(t, resp.Body, &got)
		if got.Total != 1 || got.Offset != 0 || len(got.Items) != 1 || got.Items[0].PID != pid1 {
			t.Fatalf("resources: %+v", got)
		}
	})

	t.Run("listResources_tagFilter", func(t *testing.T) {
		u := fmt.Sprintf("%s/resolver/namespaces/%s/resources?tag=alpha", base, ns)
		resp := mustGET(t, u)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)
		var got api.PaginatedResourcesResponse
		decodeJSON(t, resp.Body, &got)
		if got.Total != 1 || got.Offset != 0 || len(got.Items) != 1 || got.Items[0].Tag != "alpha" {
			t.Fatalf("filtered: %+v", got)
		}
	})

	t.Run("listResources_tagNoMatch", func(t *testing.T) {
		u := fmt.Sprintf("%s/resolver/namespaces/%s/resources?tag=other", base, ns)
		resp := mustGET(t, u)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)
		var got api.PaginatedResourcesResponse
		decodeJSON(t, resp.Body, &got)
		if got.Total != 0 || got.Offset != 0 || len(got.Items) != 0 {
			t.Fatalf("filtered: %+v", got)
		}
	})

	t.Run("listResources_tagOmitted", func(t *testing.T) {
		u := fmt.Sprintf("%s/resolver/namespaces/%s/resources", base, ns)
		resp := mustGET(t, u)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)
		var got api.PaginatedResourcesResponse
		decodeJSON(t, resp.Body, &got)
		if got.Total != 1 || got.Offset != 0 || len(got.Items) != 1 || got.Items[0].PID != pid1 {
			t.Fatalf("without tag query: want one resource, got %+v", got)
		}
	})

	t.Run("createResource_emptyTag", func(t *testing.T) {
		reqBody := mustMarshal(t, api.ResourceCreateRequest{
			URL:          "https://example.com/empty-tag",
			IdInTarget:   "empty-tag-1",
			TargetSystem: "sys-empty",
		})
		u := fmt.Sprintf("%s/resolver/namespaces/%s/resources", base, ns)
		resp := mustPOST(t, u, reqBody)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusCreated)
		var got api.ResourceResponse
		decodeJSON(t, resp.Body, &got)
		if got.Tag != "" {
			t.Fatalf("want empty tag, got %+v", got)
		}
	})

	t.Run("listResources_tagEmpty", func(t *testing.T) {
		u := fmt.Sprintf("%s/resolver/namespaces/%s/resources?tag=", base, ns)
		resp := mustGET(t, u)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)
		var got api.PaginatedResourcesResponse
		decodeJSON(t, resp.Body, &got)
		if got.Total != 1 || got.Offset != 0 || len(got.Items) != 1 || got.Items[0].Tag != "" || got.Items[0].URL != "https://example.com/empty-tag" {
			t.Fatalf("empty tag filter: %+v", got)
		}
	})

	t.Run("getResource", func(t *testing.T) {
		u := fmt.Sprintf("%s/resolver/namespaces/%s/resources/%s", base, ns, pid1)
		resp := mustGET(t, u)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)
		var got api.ResourceResponse
		decodeJSON(t, resp.Body, &got)
		if got.PID != pid1 {
			t.Fatalf("pid: %+v", got)
		}
	})

	t.Run("getResource_notFound", func(t *testing.T) {
		u := fmt.Sprintf("%s/resolver/namespaces/%s/resources/99999", base, ns)
		resp := mustGET(t, u)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusNotFound)
		assertErrorJSON(t, resp, "resource not found")
	})

	t.Run("updateResource", func(t *testing.T) {
		patch := mustMarshal(t, api.ResourceUpdateRequest{
			URL:          "https://example.com/updated",
			IdInTarget:   "ext-1b",
			TargetSystem: "sys-b",
			Tag:          "beta",
			Deleted:      false,
		})
		u := fmt.Sprintf("%s/resolver/namespaces/%s/resources/%s", base, ns, pid1)
		resp := mustPATCH(t, u, patch)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)
		var got api.ResourceResponse
		decodeJSON(t, resp.Body, &got)
		if got.URL != "https://example.com/updated" || got.IdInTarget != "ext-1b" ||
			got.TargetSystem != "sys-b" || got.Tag != "beta" || got.Deleted {
			t.Fatalf("updated: %+v", got)
		}
		if got.DateCreated == "" || got.DateUpdated == "" {
			t.Fatalf("timestamps: created=%q updated=%q", got.DateCreated, got.DateUpdated)
		}
	})

	t.Run("getResource_afterPatch", func(t *testing.T) {
		u := fmt.Sprintf("%s/resolver/namespaces/%s/resources/%s", base, ns, pid1)
		resp := mustGET(t, u)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)
		var got api.ResourceResponse
		decodeJSON(t, resp.Body, &got)
		if got.URL != "https://example.com/updated" || got.Tag != "beta" {
			t.Fatalf("resource: %+v", got)
		}
	})

	t.Run("batchCreateResources", func(t *testing.T) {
		batch := []api.ResourceCreateRequest{
			{URL: "https://b1", IdInTarget: "b1", TargetSystem: "batch", Tag: "batch"},
			{URL: "https://b2", IdInTarget: "b2", TargetSystem: "batch"},
		}
		body := mustMarshal(t, batch)
		u := fmt.Sprintf("%s/resolver/namespaces/%s/resources:batch", base, ns)
		resp := mustPOST(t, u, body)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusCreated)
		var got []api.ResourceResponse
		decodeJSON(t, resp.Body, &got)
		if len(got) != 2 || got[0].PID == "" || got[1].PID == "" {
			t.Fatalf("batch: %+v", got)
		}
	})

	t.Run("listResources_allAfterBatch", func(t *testing.T) {
		resp := mustGET(t, fmt.Sprintf("%s/resolver/namespaces/%s/resources", base, ns))
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusOK)
		var got api.PaginatedResourcesResponse
		decodeJSON(t, resp.Body, &got)
		// pid1 (updated), empty-tag create, batch b1 + b2
		if got.Total != 4 || got.Offset != 0 || len(got.Items) != 4 {
			t.Fatalf("resources: %+v", got)
		}
	})

	t.Run("batchCreate_namespaceMissing", func(t *testing.T) {
		body := mustMarshal(t, []api.ResourceCreateRequest{
			{URL: "https://x", IdInTarget: "x", TargetSystem: "y"},
		})
		u := base + "/resolver/namespaces/missing/resources:batch"
		resp := mustPOST(t, u, body)
		defer resp.Body.Close()
		assertStatus(t, resp, http.StatusNotFound)
		assertErrorJSON(t, resp, "namespace not found")
	})
}
