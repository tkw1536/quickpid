//spellchecker:words server
package server

//spellchecker:words http github quickpid
import (
	"fmt"
	"net/http"

	"github.com/tkw1536/quickpid/api"
)

func (h *Server) getNamespaceRoleScope(r *http.Request, caller *api.Caller) (api.ValidUsername, api.NamespaceScope, error) {
	username, err := h.readUsernameParam(r)
	if err != nil {
		return api.ValidUsername{}, "", err
	}
	scope := api.ScopeGetNamespaceRoleOther
	if caller != nil && username == caller.Username() {
		scope = api.ScopeGetNamespaceRoleSelf
	}
	return username, scope, nil
}

func (h *Server) getNamespaceRole(w http.ResponseWriter, r *http.Request, caller *api.Caller, namespace api.ValidNamespaceID, username api.ValidUsername) (*api.NamespaceRole, error) {
	role, err := h.svc.GetNamespaceRole(r.Context(), caller, namespace, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace role: %w", err)
	}
	return role, nil
}

func (h *Server) listNamespaceRoles(w http.ResponseWriter, r *http.Request, caller *api.Caller, namespace api.ValidNamespaceID) (*api.PaginatedNamespaceRolesResponse, error) {
	limit, offset, err := h.parsePagination(r)
	if err != nil {
		return nil, err
	}

	roles, err := h.svc.ListNamespaceRoles(r.Context(), *caller, namespace, api.ListNamespaceRolesParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespace roles: %w", err)
	}
	return roles, nil
}

func (h *Server) setNamespaceRoleScope(r *http.Request, caller *api.Caller) (api.ValidUsername, api.NamespaceScope, error) {
	username, err := h.readUsernameParam(r)
	if err != nil {
		return api.ValidUsername{}, "", err
	}
	scope := api.ScopeSetNamespaceRoleOther
	if caller != nil && username == caller.Username() {
		scope = api.ScopeSetNamespaceRoleSelf
	}
	return username, scope, nil
}

func (h *Server) setNamespaceRole(w http.ResponseWriter, r *http.Request, caller *api.Caller, namespace api.ValidNamespaceID, username api.ValidUsername) (*api.NamespaceRole, error) {
	var req api.SetNamespaceRoleRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}

	role, err := h.svc.SetNamespaceRole(r.Context(), *caller, namespace, username, req)
	if err != nil {
		return nil, fmt.Errorf("failed to set namespace role: %w", err)
	}
	return role, nil
}

func (h *Server) removeNamespaceRoleScope(r *http.Request, caller *api.Caller) (api.ValidUsername, api.NamespaceScope, error) {
	username, err := h.readUsernameParam(r)
	if err != nil {
		return api.ValidUsername{}, "", err
	}
	scope := api.ScopeRemoveNamespaceRoleOther
	if caller != nil && username == caller.Username() {
		scope = api.ScopeRemoveNamespaceRoleSelf
	}
	return username, scope, nil
}

func (h *Server) removeNamespaceRole(w http.ResponseWriter, r *http.Request, caller *api.Caller, namespace api.ValidNamespaceID, username api.ValidUsername) (struct{}, error) {
	if err := h.svc.DeleteNamespaceRole(r.Context(), *caller, namespace, username); err != nil {
		return struct{}{}, fmt.Errorf("failed to delete namespace role: %w", err)
	}
	return struct{}{}, nil
}

func (h *Server) getUserRoles(w http.ResponseWriter, r *http.Request, caller *api.Caller) (*api.PaginatedUserRolesResponse, error) {
	limit, offset, err := h.parsePagination(r)
	if err != nil {
		return nil, err
	}

	roles, err := h.svc.ListUserRoles(r.Context(), *caller, api.ListUserRolesParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list user roles: %w", err)
	}
	return roles, nil
}
