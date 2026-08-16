//spellchecker:words server
package server

//spellchecker:words context errors http strconv github quickpid
import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/tkw1536/quickpid/api"
)

func (h *Server) getResolverInfo(w http.ResponseWriter, r *http.Request) (*api.InfoResponse, error) {
	info, err := h.svc.GetResolverInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get resolver info: %w", err)
	}
	return info, nil
}

func (h *Server) getResolverMeta(w http.ResponseWriter, r *http.Request) (*api.MetaResponse, error) {
	meta := h.ops.Meta
	return &meta, nil
}

func (h *Server) listNamespaces(w http.ResponseWriter, r *http.Request, caller *api.Caller) (*api.PaginatedNamespacesResponse, error) {
	limit, offset, err := h.parsePagination(r)
	if err != nil {
		return nil, err
	}

	query := r.URL.Query()
	var tag *string
	if query.Has("tag") {
		v := query.Get("tag")
		tag = &v
	}

	namespaces, err := h.svc.ListNamespaces(r.Context(), caller, api.ListNamespacesParams{
		Tag:    tag,
		Limit:  limit,
		Offset: offset,
	}, func(ctx context.Context, namespace api.ValidNamespaceID) bool {
		return h.lowlevel.CheckNamespaceScope(r, namespace, caller, api.ScopeReadMetadata) == nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}
	return namespaces, nil
}

func (h *Server) getNamespace(w http.ResponseWriter, r *http.Request, caller *api.Caller, namespace api.ValidNamespaceID) (*api.NamespaceResponse, error) {
	res, err := h.svc.GetNamespace(r.Context(), caller, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace: %w", err)
	}
	return res, nil
}

func (h *Server) updateNamespace(w http.ResponseWriter, r *http.Request, caller *api.Caller, namespace api.ValidNamespaceID) (*api.NamespaceResponse, error) {
	var req api.NamespaceUpdateRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}

	validReq, err := req.Validate()
	if err != nil {
		return nil, fmt.Errorf("failed to validate namespace update request: %w", err)
	}

	res, err := h.svc.UpdateNamespace(r.Context(), caller, namespace, validReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update namespace: %w", err)
	}
	return res, nil
}

func (h *Server) countAllResources(w http.ResponseWriter, r *http.Request, caller *api.Caller) (*api.ResourceCountResponse, error) {
	count, err := h.svc.CountAllResources(r.Context(), caller)
	if err != nil {
		return nil, fmt.Errorf("failed to count all resources: %w", err)
	}
	return count, nil
}

func (h *Server) createNamespace(w http.ResponseWriter, r *http.Request, caller *api.Caller) (*api.NamespaceResponse, error) {
	var req api.NamespaceCreateRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}
	namespace, err := h.svc.CreateNamespace(r.Context(), caller, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create namespace: %w", err)
	}
	return namespace, nil
}

func (h *Server) listResources(w http.ResponseWriter, r *http.Request, caller *api.Caller, namespace api.ValidNamespaceID) (*api.PaginatedResourcesResponse, error) {
	query := r.URL.Query()

	var tag *string
	if query.Has("tag") {
		v := query.Get("tag")
		tag = &v
	}

	var deleted *bool
	if query.Has("deleted") {
		b, err := strconv.ParseBool(query.Get("deleted"))
		if err != nil {
			return nil, api.WithErrorCode(errors.Join(errDeletedInvalid, err), api.InvalidQueryParameter)
		}
		deleted = &b
	}

	limit, offset, err := h.parsePagination(r)
	if err != nil {
		return nil, err
	}

	resources, err := h.svc.ListResources(r.Context(), caller, namespace, api.ListResourcesParams{
		Tag:     tag,
		Deleted: deleted,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}
	return resources, nil
}

func (h *Server) createResource(w http.ResponseWriter, r *http.Request, caller *api.Caller, namespace api.ValidNamespaceID) (*api.ResourceResponse, error) {
	var req api.ResourceCreateRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}

	validReq, err := req.Validate()
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("failed to validate resource create request: %w", err), api.InvalidResourceURL)
	}

	resource, err := h.svc.CreateResource(r.Context(), caller, namespace, validReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}
	return resource, nil
}

func (h *Server) createResourceBatch(w http.ResponseWriter, r *http.Request, caller *api.Caller, namespace api.ValidNamespaceID) ([]api.ResourceResponse, error) {
	var reqs []api.ResourceCreateRequest
	if err := h.decodeJSON(w, r, &reqs); err != nil {
		return nil, err
	}

	validReqs := make([]api.ValidResourceCreateRequest, len(reqs))
	for i := range reqs {
		validReq, err := reqs[i].Validate()
		if err != nil {
			return nil, api.WithErrorCode(fmt.Errorf("failed to validate resource create request: %w", err), api.InvalidResourceURL)
		}
		validReqs[i] = validReq
	}

	resources, err := h.svc.BatchCreateResources(r.Context(), caller, namespace, validReqs)
	if err != nil {
		return nil, fmt.Errorf("failed to batch create resources: %w", err)
	}
	return resources, nil
}

// shouldRedactResource returns a function that can check if the caller should see a redacted resource in the given namespace.
func (h *Server) shouldRedactResource(r *http.Request, caller *api.Caller, namespace api.ValidNamespaceID) func() bool {
	return func() bool {
		return h.lowlevel.CheckNamespaceScope(r, namespace, caller, api.ScopeSeeDeletedResource) != nil
	}
}

func (h *Server) getResource(w http.ResponseWriter, r *http.Request, caller *api.Caller, namespace api.ValidNamespaceID) (api.ResourceGetResult, error) {
	resourcePID, err := h.readPidParam(r)
	if err != nil {
		return nil, err
	}

	resource, err := h.svc.GetResource(r.Context(), caller, namespace, resourcePID, h.shouldRedactResource(r, caller, namespace))
	if err != nil {
		return nil, fmt.Errorf("failed to get resource: %w", err)
	}
	return resource, nil
}

func (h *Server) updateResource(w http.ResponseWriter, r *http.Request, caller *api.Caller, namespace api.ValidNamespaceID) (*api.ResourceResponse, error) {
	var req api.ResourceUpdateRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}

	validReq, err := req.Validate()
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("failed to validate resource update request: %w", err), api.InvalidResourceURL)
	}

	resourcePID, err := h.readPidParam(r)
	if err != nil {
		return nil, err
	}

	resource, err := h.svc.UpdateResource(r.Context(), caller, namespace, resourcePID, validReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update resource: %w", err)
	}
	return resource, nil
}
