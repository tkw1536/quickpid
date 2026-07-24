//spellchecker:words server
package server

//spellchecker:words http github quickpid
import (
	"fmt"
	"net/http"

	"github.com/tkw1536/quickpid/api"
)

func (h *Server) getCurrentUserHTTP(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (*api.UserInfo, error) {
	target, err := h.parseOptionalUsernameQuery(r)
	if err != nil {
		return nil, err
	}
	currentUser, err := h.svc.GetUserAccount(r.Context(), user, target)
	if err != nil {
		return nil, fmt.Errorf("svc.GetUserAccount: %w", err)
	}
	return currentUser, nil
}

func (h *Server) createUser(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (*api.UserInfo, error) {
	var req api.UserCreateRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}

	validReq, err := req.Validate()
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("failed to validate user create request: %w", err), api.InvalidUsername)
	}

	createdUser, err := h.svc.CreateUser(r.Context(), user, validReq)
	if err != nil {
		return nil, fmt.Errorf("svc.CreateUser: %w", err)
	}
	return createdUser, nil
}

func (h *Server) setUserPassword(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (*api.SetPasswordResponse, error) {
	target, err := h.parseOptionalUsernameQuery(r)
	if err != nil {
		return nil, err
	}
	var req api.SetPasswordRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}

	validReq, err := req.Validate()
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("failed to validate set password request: %w", err), api.InvalidPassword)
	}

	response, err := h.svc.SetUserPassword(r.Context(), user, target, validReq)
	if err != nil {
		return nil, fmt.Errorf("svc.SetUserPassword: %w", err)
	}
	return response, nil
}

func (h *Server) listUsers(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (*api.PaginatedUsersResponse, error) {
	superuser, err := h.parseOptionalBooleanQuery(r, "superuser")
	if err != nil {
		return nil, err
	}
	password, err := h.parseOptionalBooleanQuery(r, "password")
	if err != nil {
		return nil, err
	}
	limit, offset, err := h.parsePagination(r)
	if err != nil {
		return nil, err
	}

	users, err := h.svc.ListUsers(r.Context(), user, api.ListUsersParams{
		Superuser: superuser,
		Password:  password,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, fmt.Errorf("svc.ListUsers: %w", err)
	}
	return users, nil
}

func (h *Server) autocompleteUsers(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) ([]string, error) {
	query, err := h.parseRequiredAutocompleteQuery(r)
	if err != nil {
		return nil, err
	}
	usernames, err := h.svc.AutocompleteUsers(r.Context(), user, query.String())
	if err != nil {
		return nil, fmt.Errorf("svc.AutocompleteUsers: %w", err)
	}
	return usernames, nil
}

func (h *Server) listKeys(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (*api.PaginatedAPIKeysResponse, error) {
	target, err := h.parseOptionalUsernameQuery(r)
	if err != nil {
		return nil, err
	}
	limit, offset, err := h.parsePagination(r)
	if err != nil {
		return nil, err
	}

	keys, err := h.svc.ListKeys(r.Context(), user, target, api.ListKeysParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("svc.ListKeys: %w", err)
	}
	return keys, nil
}

func (h *Server) issueKey(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (*api.IssueKeyResponse, error) {
	target, err := h.parseOptionalUsernameQuery(r)
	if err != nil {
		return nil, err
	}
	var req api.KeyIssueRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}

	response, err := h.svc.IssueKey(r.Context(), user, target, req)
	if err != nil {
		return nil, fmt.Errorf("svc.IssueKey: %w", err)
	}
	return response, nil
}

func (h *Server) revokeKey(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (*api.RevokeKeyResponse, error) {
	target, err := h.parseOptionalUsernameQuery(r)
	if err != nil {
		return nil, err
	}
	var req api.KeyRevokeRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}

	response, err := h.svc.RevokeKey(r.Context(), user, target, req)
	if err != nil {
		return nil, fmt.Errorf("svc.RevokeKey: %w", err)
	}
	return response, nil
}

func (h *Server) updateCurrentUser(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (*api.UserInfo, error) {
	target, err := h.parseRequiredUsernameQuery(r)
	if err != nil {
		return nil, err
	}
	var req api.UserUpdateRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}

	updatedUser, err := h.svc.UpdateUser(r.Context(), user, target, req)
	if err != nil {
		return nil, fmt.Errorf("svc.UpdateUser: %w", err)
	}
	return updatedUser, nil
}
