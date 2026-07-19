//spellchecker:words server
package server

//spellchecker:words http github quickpid
import (
	"fmt"
	"net/http"

	"github.com/tkw1536/quickpid/api"
)

func (h *Server) getNamespacePermission(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (*api.NamespacePermission, error) {
	namespace, err := h.getNamespace(r)
	if err != nil {
		return nil, err
	}
	username, err := h.getUsername(r)
	if err != nil {
		return nil, err
	}
	permission, err := h.svc.GetNamespacePermission(r.Context(), user, namespace, username)
	if err != nil {
		return nil, fmt.Errorf("svc.GetNamespacePermission: %w", err)
	}
	return permission, nil
}

func (h *Server) listNamespacePermissions(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (*api.PaginatedNamespacePermissionsResponse, error) {
	namespace, err := h.getNamespace(r)
	if err != nil {
		return nil, err
	}

	limit, offset, err := h.parsePagination(r)
	if err != nil {
		return nil, err
	}

	permissions, err := h.svc.ListNamespacePermissions(r.Context(), user, namespace, api.ListNamespacePermissionsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("svc.ListNamespacePermissions: %w", err)
	}
	return permissions, nil
}

func (h *Server) setNamespacePermission(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (*api.NamespacePermission, error) {
	namespace, err := h.getNamespace(r)
	if err != nil {
		return nil, err
	}
	username, err := h.getUsername(r)
	if err != nil {
		return nil, err
	}

	var req api.SetNamespacePermissionRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}

	permission, err := h.svc.SetNamespacePermission(r.Context(), user, namespace, username, req)
	if err != nil {
		return nil, fmt.Errorf("svc.SetNamespacePermission: %w", err)
	}
	return permission, nil
}

func (h *Server) deleteNamespacePermission(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (struct{}, error) {
	namespace, err := h.getNamespace(r)
	if err != nil {
		return struct{}{}, err
	}
	username, err := h.getUsername(r)
	if err != nil {
		return struct{}{}, err
	}

	if err := h.svc.DeleteNamespacePermission(r.Context(), user, namespace, username); err != nil {
		return struct{}{}, fmt.Errorf("svc.DeleteNamespacePermission: %w", err)
	}
	return struct{}{}, nil
}
