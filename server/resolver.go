//spellchecker:words server
package server

//spellchecker:words errors http strconv github bicpid
import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/tkw1536/bicpid/api"
)

func (h *Server) getResolverInfo(w http.ResponseWriter, r *http.Request) (*api.InfoResponse, api.Error, error) {
	info, specError, err := h.svc.GetResolverInfo()
	if err != nil {
		return nil, specError, fmt.Errorf("store.GetResolverInfo: %w", err)
	}
	return info, "", nil
}

func (h *Server) listNamespaces(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (*api.PaginatedNamespacesResponse, api.Error, error) {
	limit, offset, specError, err := h.parsePagination(r)
	if err != nil {
		return nil, specError, err
	}

	query := r.URL.Query()
	var tag *string
	if query.Has("tag") {
		v := query.Get("tag")
		tag = &v
	}

	namespaces, specError, err := h.svc.ListNamespaces(r.Context(), user, api.ListNamespacesParams{
		Tag:    tag,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, specError, fmt.Errorf("store.ListNamespaces: %w", err)
	}
	return namespaces, "", nil
}

func (h *Server) getNamespaceDetail(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (*api.NamespaceResponse, api.Error, error) {
	namespace, specError, err := h.getNamespace(r)
	if err != nil {
		return nil, specError, err
	}
	res, specError, err := h.svc.GetNamespace(r.Context(), user, namespace)
	if err != nil {
		return nil, specError, fmt.Errorf("store.GetNamespace: %w", err)
	}
	return res, "", nil
}

func (h *Server) countAllResources(w http.ResponseWriter, r *http.Request) (*api.ResourceCountResponse, api.Error, error) {
	count, specError, err := h.svc.CountAllResources(r.Context())
	if err != nil {
		return nil, specError, fmt.Errorf("store.CountAllResources: %w", err)
	}
	return count, "", nil
}

func (h *Server) createNamespace(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (*api.NamespaceResponse, api.Error, error) {
	var req api.NamespaceCreateRequest
	if specError, err := h.decodeJSON(w, r, &req); err != nil {
		return nil, specError, err
	}
	namespace, specError, err := h.svc.CreateNamespace(r.Context(), user, req)
	if err != nil {
		return nil, specError, fmt.Errorf("store.CreateNamespace: %w", err)
	}
	return namespace, "", nil
}

func (h *Server) listResources(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (*api.PaginatedResourcesResponse, api.Error, error) {
	namespace, specError, err := h.getNamespace(r)
	if err != nil {
		return nil, specError, err
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
			return nil, api.InvalidQueryParameter, errors.Join(errDeletedInvalid, err)
		}
		deleted = &b
	}

	limit, offset, specError, err := h.parsePagination(r)
	if err != nil {
		return nil, specError, err
	}

	resources, specError, err := h.svc.ListResources(r.Context(), user, api.ListResourcesParams{
		Namespace: namespace,
		Tag:       tag,
		Deleted:   deleted,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, specError, fmt.Errorf("store.ListResources: %w", err)
	}
	return resources, "", nil
}

func (h *Server) createResource(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (*api.ResourceResponse, api.Error, error) {
	var req api.ResourceCreateRequest
	if specError, err := h.decodeJSON(w, r, &req); err != nil {
		return nil, specError, err
	}

	namespace, specError, err := h.getNamespace(r)
	if err != nil {
		return nil, specError, err
	}

	resource, specError, err := h.svc.CreateResource(r.Context(), user, namespace, req)
	if err != nil {
		return nil, specError, fmt.Errorf("store.CreateResource: %w", err)
	}
	return resource, "", nil
}

func (h *Server) batchCreateResources(w http.ResponseWriter, r *http.Request, user *api.UserInfo) ([]api.ResourceResponse, api.Error, error) {
	var reqs []api.ResourceCreateRequest
	if specError, err := h.decodeJSON(w, r, &reqs); err != nil {
		return nil, specError, err
	}

	namespace, specError, err := h.getNamespace(r)
	if err != nil {
		return nil, specError, err
	}

	resources, specError, err := h.svc.BatchCreateResources(r.Context(), user, namespace, reqs)
	if err != nil {
		return nil, specError, fmt.Errorf("store.BatchCreateResources: %w", err)
	}
	return resources, "", nil
}

func (h *Server) getResource(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (*api.ResourceResponse, api.Error, error) {
	namespace, specError, err := h.getNamespace(r)
	if err != nil {
		return nil, specError, err
	}
	resourcePID, specError, err := h.getPID(r)
	if err != nil {
		return nil, specError, err
	}

	resource, specError, err := h.svc.GetResource(r.Context(), user, namespace, resourcePID)
	if err != nil {
		return nil, specError, fmt.Errorf("store.GetResource: %w", err)
	}
	return resource, "", nil
}

func (h *Server) updateResource(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (*api.ResourceResponse, api.Error, error) {
	var req api.ResourceUpdateRequest
	if specError, err := h.decodeJSON(w, r, &req); err != nil {
		return nil, specError, err
	}
	namespace, specError, err := h.getNamespace(r)
	if err != nil {
		return nil, specError, err
	}

	resourcePID, specError, err := h.getPID(r)
	if err != nil {
		return nil, specError, err
	}

	resource, specError, err := h.svc.UpdateResource(r.Context(), user, namespace, resourcePID, req)
	if err != nil {
		return nil, specError, fmt.Errorf("store.UpdateResource: %w", err)
	}
	return resource, "", nil
}
