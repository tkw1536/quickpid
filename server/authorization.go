//spellchecker:words server
package server

//spellchecker:words http github quickpid
import (
	"fmt"
	"net/http"

	"github.com/tkw1536/quickpid/api"
)

func (h *Server) getNamespaceRole(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (*api.NamespaceRole, error) {
	namespace, err := h.getNamespace(r)
	if err != nil {
		return nil, err
	}
	username, err := h.getUsername(r)
	if err != nil {
		return nil, err
	}
	role, err := h.svc.GetNamespaceRole(r.Context(), user, namespace, username)
	if err != nil {
		return nil, fmt.Errorf("svc.GetNamespaceRole: %w", err)
	}
	return role, nil
}

func (h *Server) listNamespaceRoles(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (*api.PaginatedNamespaceRolesResponse, error) {
	namespace, err := h.getNamespace(r)
	if err != nil {
		return nil, err
	}

	limit, offset, err := h.parsePagination(r)
	if err != nil {
		return nil, err
	}

	roles, err := h.svc.ListNamespaceRoles(r.Context(), user, namespace, api.ListNamespaceRolesParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("svc.ListNamespaceRoles: %w", err)
	}
	return roles, nil
}

func (h *Server) setNamespaceRole(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (*api.NamespaceRole, error) {
	namespace, err := h.getNamespace(r)
	if err != nil {
		return nil, err
	}
	username, err := h.getUsername(r)
	if err != nil {
		return nil, err
	}

	var req api.SetNamespaceRoleRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}

	role, err := h.svc.SetNamespaceRole(r.Context(), user, namespace, username, req)
	if err != nil {
		return nil, fmt.Errorf("svc.SetNamespaceRole: %w", err)
	}
	return role, nil
}

func (h *Server) deleteNamespaceRole(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (struct{}, error) {
	namespace, err := h.getNamespace(r)
	if err != nil {
		return struct{}{}, err
	}
	username, err := h.getUsername(r)
	if err != nil {
		return struct{}{}, err
	}

	if err := h.svc.DeleteNamespaceRole(r.Context(), user, namespace, username); err != nil {
		return struct{}{}, fmt.Errorf("svc.DeleteNamespaceRole: %w", err)
	}
	return struct{}{}, nil
}

func (h *Server) listUserRoles(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (*api.PaginatedUserRolesResponse, error) {
	target, err := h.parseOptionalUsernameQuery(r)
	if err != nil {
		return nil, err
	}
	limit, offset, err := h.parsePagination(r)
	if err != nil {
		return nil, err
	}

	roles, err := h.svc.ListUserRoles(r.Context(), user, target, api.ListUserRolesParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("svc.ListUserRoles: %w", err)
	}
	return roles, nil
}
