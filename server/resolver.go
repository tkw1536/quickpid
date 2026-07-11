//spellchecker:words server
package server

//spellchecker:words errors http strconv github bicpid
import (
	"errors"
	"net/http"
	"strconv"

	"github.com/tkw1536/bicpid/api"
)

func (h *Server) getResolverInfo(w http.ResponseWriter, r *http.Request) (*api.InfoResponse, api.Error, error) {
	return h.svc.GetResolverInfo()
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

	return h.svc.ListNamespaces(r.Context(), user, api.ListNamespacesParams{
		Tag:    tag,
		Limit:  limit,
		Offset: offset,
	})
}

func (h *Server) getNamespaceDetail(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (*api.NamespaceResponse, api.Error, error) {
	namespace, specError, err := h.getNamespace(r)
	if err != nil {
		return nil, specError, err
	}
	return h.svc.GetNamespace(r.Context(), user, namespace)
}

func (h *Server) countAllResources(w http.ResponseWriter, r *http.Request) (*api.ResourceCountResponse, api.Error, error) {
	return h.svc.CountAllResources(r.Context())
}

func (h *Server) createNamespace(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (*api.NamespaceResponse, api.Error, error) {
	var req api.NamespaceCreateRequest
	if specError, err := h.decodeJSON(w, r, &req); err != nil {
		return nil, specError, err
	}
	return h.svc.CreateNamespace(r.Context(), user, req)
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

	return h.svc.ListResources(r.Context(), user, api.ListResourcesParams{
		Namespace: namespace,
		Tag:       tag,
		Deleted:   deleted,
		Limit:     limit,
		Offset:    offset,
	})
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

	return h.svc.CreateResource(r.Context(), user, namespace, req)
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

	return h.svc.BatchCreateResources(r.Context(), user, namespace, reqs)
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

	return h.svc.GetResource(r.Context(), user, namespace, resourcePID)
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

	return h.svc.UpdateResource(r.Context(), user, namespace, resourcePID, req)
}
