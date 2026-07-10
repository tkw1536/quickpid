//spellchecker:words server
package server

//spellchecker:words http github bicpid
import (
	"net/http"

	"github.com/tkw1536/bicpid/api"
)

func (h *Server) getNamespacePermission(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (*api.NamespacePermission, api.Error, error) {
	namespace, specError, err := h.getNamespace(r)
	if err != nil {
		return nil, specError, err
	}
	username, specError, err := h.getUsername(r)
	if err != nil {
		return nil, specError, err
	}
	return h.svc.GetNamespacePermission(r.Context(), user, namespace, username)
}

func (h *Server) listNamespacePermissions(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (*api.PaginatedNamespacePermissionsResponse, api.Error, error) {
	namespace, specError, err := h.getNamespace(r)
	if err != nil {
		return nil, specError, err
	}

	limit, offset, specError, err := h.parsePagination(r)
	if err != nil {
		return nil, specError, err
	}

	return h.svc.ListNamespacePermissions(r.Context(), user, namespace, api.ListNamespacePermissionsParams{
		Limit:  limit,
		Offset: offset,
	})
}

func (h *Server) setNamespacePermission(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (*api.NamespacePermission, api.Error, error) {
	namespace, specError, err := h.getNamespace(r)
	if err != nil {
		return nil, specError, err
	}
	username, specError, err := h.getUsername(r)
	if err != nil {
		return nil, specError, err
	}

	var req api.SetNamespacePermissionRequest
	if specError, err := h.decodeJSON(w, r, &req); err != nil {
		return nil, specError, err
	}

	return h.svc.SetNamespacePermission(r.Context(), user, namespace, username, req)
}

func (h *Server) deleteNamespacePermission(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (struct{}, api.Error, error) {
	namespace, specError, err := h.getNamespace(r)
	if err != nil {
		return struct{}{}, specError, err
	}
	username, specError, err := h.getUsername(r)
	if err != nil {
		return struct{}{}, specError, err
	}

	specError, err = h.svc.DeleteNamespacePermission(r.Context(), user, namespace, username)
	return struct{}{}, specError, err
}
