//spellchecker:words server
package server

//spellchecker:words http github bicpid
import (
	"fmt"
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
	permission, specError, err := h.svc.GetNamespacePermission(r.Context(), user, namespace, username)
	if err != nil {
		return nil, specError, fmt.Errorf("svc.GetNamespacePermission: %w", err)
	}
	return permission, "", nil
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

	permissions, specError, err := h.svc.ListNamespacePermissions(r.Context(), user, namespace, api.ListNamespacePermissionsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, specError, fmt.Errorf("svc.ListNamespacePermissions: %w", err)
	}
	return permissions, "", nil
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

	permission, specError, err := h.svc.SetNamespacePermission(r.Context(), user, namespace, username, req)
	if err != nil {
		return nil, specError, fmt.Errorf("svc.SetNamespacePermission: %w", err)
	}
	return permission, "", nil
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
	if err != nil {
		return struct{}{}, specError, fmt.Errorf("svc.DeleteNamespacePermission: %w", err)
	}
	return struct{}{}, "", nil
}
