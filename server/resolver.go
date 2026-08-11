//spellchecker:words server
package server

//spellchecker:words errors http strconv github quickpid
import (
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

func (h *Server) listNamespaces(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (*api.PaginatedNamespacesResponse, error) {
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

	namespaces, err := h.svc.ListNamespaces(r.Context(), user, api.ListNamespacesParams{
		Tag:    tag,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}
	return namespaces, nil
}

func (h *Server) getNamespace(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (*api.NamespaceResponse, error) {
	namespace, err := h.readNamespaceParam(r)
	if err != nil {
		return nil, err
	}
	res, err := h.svc.GetNamespace(r.Context(), user, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace: %w", err)
	}
	return res, nil
}

func (h *Server) countAllResources(w http.ResponseWriter, r *http.Request) (*api.ResourceCountResponse, error) {
	count, err := h.svc.CountAllResources(r.Context())
	if err != nil {
		return nil, fmt.Errorf("failed to count all resources: %w", err)
	}
	return count, nil
}

func (h *Server) createNamespace(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (*api.NamespaceResponse, error) {
	var req api.NamespaceCreateRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}
	namespace, err := h.svc.CreateNamespace(r.Context(), user, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create namespace: %w", err)
	}
	return namespace, nil
}

func (h *Server) listResources(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (*api.PaginatedResourcesResponse, error) {
	namespace, err := h.readNamespaceParam(r)
	if err != nil {
		return nil, err
	}
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

	resources, err := h.svc.ListResources(r.Context(), user, namespace, api.ListResourcesParams{
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

func (h *Server) createResource(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (*api.ResourceResponse, error) {
	var req api.ResourceCreateRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}

	namespace, err := h.readNamespaceParam(r)
	if err != nil {
		return nil, err
	}

	resource, err := h.svc.CreateResource(r.Context(), user, namespace, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}
	return resource, nil
}

func (h *Server) createResourceBatch(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) ([]api.ResourceResponse, error) {
	var reqs []api.ResourceCreateRequest
	if err := h.decodeJSON(w, r, &reqs); err != nil {
		return nil, err
	}

	namespace, err := h.readNamespaceParam(r)
	if err != nil {
		return nil, err
	}

	resources, err := h.svc.BatchCreateResources(r.Context(), user, namespace, reqs)
	if err != nil {
		return nil, fmt.Errorf("failed to batch create resources: %w", err)
	}
	return resources, nil
}

func (h *Server) getResource(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (api.ResourceGetResult, error) {
	namespace, err := h.readNamespaceParam(r)
	if err != nil {
		return nil, err
	}
	resourcePID, err := h.readPidParam(r)
	if err != nil {
		return nil, err
	}

	resource, err := h.svc.GetResource(r.Context(), user, namespace, resourcePID)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource: %w", err)
	}
	return resource, nil
}

func (h *Server) updateResource(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (*api.ResourceResponse, error) {
	var req api.ResourceUpdateRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}
	namespace, err := h.readNamespaceParam(r)
	if err != nil {
		return nil, err
	}

	resourcePID, err := h.readPidParam(r)
	if err != nil {
		return nil, err
	}

	resource, err := h.svc.UpdateResource(r.Context(), user, namespace, resourcePID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update resource: %w", err)
	}
	return resource, nil
}
