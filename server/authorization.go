//spellchecker:words server
package server

//spellchecker:words http github quickpid
import (
	"fmt"
	"net/http"

	"github.com/tkw1536/quickpid/api"
)

func (h *Server) getNamespaceRole(w http.ResponseWriter, r *http.Request, user api.ValidUserInfo) (*api.NamespaceRole, error) {
	namespace, err := h.readNamespaceParam(r)
	if err != nil {
		return nil, err
	}
	username, err := h.readUsernameParam(r)
	if err != nil {
		return nil, err
	}
	role, err := h.svc.GetNamespaceRole(r.Context(), user, namespace, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace role: %w", err)
	}
	return role, nil
}

func (h *Server) listNamespaceRoles(w http.ResponseWriter, r *http.Request, user api.ValidUserInfo) (*api.PaginatedNamespaceRolesResponse, error) {
	namespace, err := h.readNamespaceParam(r)
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
		return nil, fmt.Errorf("failed to list namespace roles: %w", err)
	}
	return roles, nil
}

func (h *Server) setNamespaceRole(w http.ResponseWriter, r *http.Request, user api.ValidUserInfo) (*api.NamespaceRole, error) {
	namespace, err := h.readNamespaceParam(r)
	if err != nil {
		return nil, err
	}
	username, err := h.readUsernameParam(r)
	if err != nil {
		return nil, err
	}

	var req api.SetNamespaceRoleRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}

	role, err := h.svc.SetNamespaceRole(r.Context(), user, namespace, username, req)
	if err != nil {
		return nil, fmt.Errorf("failed to set namespace role: %w", err)
	}
	return role, nil
}

func (h *Server) removeNamespaceRole(w http.ResponseWriter, r *http.Request, caller api.ValidUserInfo) (struct{}, error) {
	namespace, err := h.readNamespaceParam(r)
	if err != nil {
		return struct{}{}, err
	}
	username, err := h.readUsernameParam(r)
	if err != nil {
		return struct{}{}, err
	}

	if err := h.svc.DeleteNamespaceRole(r.Context(), caller, namespace, username); err != nil {
		return struct{}{}, fmt.Errorf("failed to delete namespace role: %w", err)
	}
	return struct{}{}, nil
}

func (h *Server) getUserRoles(w http.ResponseWriter, r *http.Request, user api.ValidUserInfo) (*api.PaginatedUserRolesResponse, error) {
	limit, offset, err := h.parsePagination(r)
	if err != nil {
		return nil, err
	}

	roles, err := h.svc.ListUserRoles(r.Context(), user, api.ListUserRolesParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list user roles: %w", err)
	}
	return roles, nil
}
