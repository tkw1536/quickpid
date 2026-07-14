//spellchecker:words server
package server

//spellchecker:words http github bicpid
import (
	"fmt"
	"net/http"

	"github.com/tkw1536/bicpid/api"
)

func (h *Server) getCurrentUserHTTP(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (*api.UserInfo, api.Error, error) {
	target, specError, err := h.parseOptionalUsernameQuery(r)
	if err != nil {
		return nil, specError, err
	}
	currentUser, specError, err := h.svc.GetUserAccount(r.Context(), user, target)
	if err != nil {
		return nil, specError, fmt.Errorf("svc.GetUserAccount: %w", err)
	}
	return currentUser, "", nil
}

func (h *Server) createUser(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (*api.UserInfo, api.Error, error) {
	var req api.UserCreateRequest
	if specError, err := h.decodeJSON(w, r, &req); err != nil {
		return nil, specError, err
	}

	createdUser, specError, err := h.svc.CreateUser(r.Context(), user, req)
	if err != nil {
		return nil, specError, fmt.Errorf("svc.CreateUser: %w", err)
	}
	return createdUser, "", nil
}

func (h *Server) listUsers(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (*api.PaginatedUsersResponse, api.Error, error) {
	superuser, specError, err := h.parseSuperuserQuery(r)
	if err != nil {
		return nil, specError, err
	}
	limit, offset, specError, err := h.parsePagination(r)
	if err != nil {
		return nil, specError, err
	}

	users, specError, err := h.svc.ListUsers(r.Context(), user, api.ListUsersParams{
		Superuser: superuser,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, specError, fmt.Errorf("svc.ListUsers: %w", err)
	}
	return users, "", nil
}

func (h *Server) autocompleteUsers(w http.ResponseWriter, r *http.Request, user *api.UserInfo) ([]string, api.Error, error) {
	query, specError, err := h.parseRequiredAutocompleteQuery(r)
	if err != nil {
		return nil, specError, err
	}
	usernames, specError, err := h.svc.AutocompleteUsers(r.Context(), user, query)
	if err != nil {
		return nil, specError, fmt.Errorf("svc.AutocompleteUsers: %w", err)
	}
	return usernames, "", nil
}

func (h *Server) listKeys(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (*api.PaginatedAPIKeysResponse, api.Error, error) {
	target, specError, err := h.parseOptionalUsernameQuery(r)
	if err != nil {
		return nil, specError, err
	}
	limit, offset, specError, err := h.parsePagination(r)
	if err != nil {
		return nil, specError, err
	}

	keys, specError, err := h.svc.ListKeys(r.Context(), user, target, api.ListKeysParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, specError, fmt.Errorf("svc.ListKeys: %w", err)
	}
	return keys, "", nil
}

func (h *Server) issueKey(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (*api.IssueKeyResponse, api.Error, error) {
	target, specError, err := h.parseOptionalUsernameQuery(r)
	if err != nil {
		return nil, specError, err
	}
	var req api.KeyIssueRequest
	if specError, err := h.decodeJSON(w, r, &req); err != nil {
		return nil, specError, err
	}

	response, specError, err := h.svc.IssueKey(r.Context(), user, target, req)
	if err != nil {
		return nil, specError, fmt.Errorf("svc.IssueKey: %w", err)
	}
	return response, "", nil
}

func (h *Server) revokeKey(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (*api.RevokeKeyResponse, api.Error, error) {
	target, specError, err := h.parseOptionalUsernameQuery(r)
	if err != nil {
		return nil, specError, err
	}
	var req api.KeyRevokeRequest
	if specError, err := h.decodeJSON(w, r, &req); err != nil {
		return nil, specError, err
	}

	response, specError, err := h.svc.RevokeKey(r.Context(), user, target, req)
	if err != nil {
		return nil, specError, fmt.Errorf("svc.RevokeKey: %w", err)
	}
	return response, "", nil
}

func (h *Server) updateCurrentUser(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (*api.UserInfo, api.Error, error) {
	target, specError, err := h.parseRequiredUsernameQuery(r)
	if err != nil {
		return nil, specError, err
	}
	var req api.UserUpdateRequest
	if specError, err := h.decodeJSON(w, r, &req); err != nil {
		return nil, specError, err
	}

	updatedUser, specError, err := h.svc.UpdateUser(r.Context(), user, target, req)
	if err != nil {
		return nil, specError, fmt.Errorf("svc.UpdateUser: %w", err)
	}
	return updatedUser, "", nil
}
