//spellchecker:words server
package server

//spellchecker:words http github quickpid
import (
	"fmt"
	"net/http"

	"github.com/tkw1536/quickpid/api"
)

func (h *Server) getCurrentUserHTTP(w http.ResponseWriter, r *http.Request, caller api.ValidUserInfo) (*api.UserInfo, error) {
	return &api.UserInfo{
		Username:  caller.Username.String(),
		Superuser: caller.Superuser,
		Password:  caller.Password,
	}, nil
}

func (h *Server) createUser(w http.ResponseWriter, r *http.Request, caller api.ValidUserInfo) (*api.UserInfo, error) {
	var req api.UserCreateRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}

	validReq, err := req.Validate()
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("failed to validate user create request: %w", err), api.InvalidUsername)
	}

	createdUser, err := h.svc.CreateUser(r.Context(), caller, validReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return createdUser, nil
}

func (h *Server) deleteUser(w http.ResponseWriter, r *http.Request, caller api.ValidUserInfo) (struct{}, error) {
	target, err := h.parseRequiredUsernameQuery(r)
	if err != nil {
		return struct{}{}, err
	}

	if err := h.svc.DeleteUser(r.Context(), caller, target); err != nil {
		return struct{}{}, fmt.Errorf("failed to delete user: %w", err)
	}
	return struct{}{}, nil
}

func (h *Server) setUserPassword(w http.ResponseWriter, r *http.Request, caller api.ValidUserInfo) (*api.SetPasswordResponse, error) {
	var req api.SetPasswordRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}

	validReq, err := req.Validate()
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("failed to validate set password request: %w", err), api.InvalidPassword)
	}

	response, err := h.svc.SetUserPassword(r.Context(), caller, validReq)
	if err != nil {
		return nil, fmt.Errorf("failed to set user password: %w", err)
	}
	return response, nil
}

func (h *Server) listUsers(w http.ResponseWriter, r *http.Request, caller api.ValidUserInfo) (*api.PaginatedUsersResponse, error) {
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

	users, err := h.svc.ListUsers(r.Context(), caller, api.ListUsersParams{
		Superuser: superuser,
		Password:  password,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	return users, nil
}

func (h *Server) autocompleteUsers(w http.ResponseWriter, r *http.Request, caller api.ValidUserInfo) ([]string, error) {
	query, err := h.parseRequiredAutocompleteQuery(r)
	if err != nil {
		return nil, err
	}
	usernames, err := h.svc.AutocompleteUsers(r.Context(), caller, query)
	if err != nil {
		return nil, fmt.Errorf("failed to autocomplete users: %w", err)
	}
	return usernames, nil
}

func (h *Server) listKeys(w http.ResponseWriter, r *http.Request, caller api.ValidUserInfo) (*api.PaginatedAPIKeysResponse, error) {
	limit, offset, err := h.parsePagination(r)
	if err != nil {
		return nil, err
	}

	keys, err := h.svc.ListKeys(r.Context(), caller, api.ListKeysParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}
	return keys, nil
}

func (h *Server) issueKey(w http.ResponseWriter, r *http.Request, caller api.ValidUserInfo) (*api.IssueKeyResponse, error) {
	var req api.KeyIssueRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}

	response, err := h.svc.IssueKey(r.Context(), caller, req)
	if err != nil {
		return nil, fmt.Errorf("failed to issue key: %w", err)
	}
	return response, nil
}

func (h *Server) revokeKey(w http.ResponseWriter, r *http.Request, caller api.ValidUserInfo) (struct{}, error) {
	var req api.KeyRevokeRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return struct{}{}, err
	}

	if err := h.svc.RevokeKey(r.Context(), caller, req); err != nil {
		return struct{}{}, fmt.Errorf("failed to revoke key: %w", err)
	}
	return struct{}{}, nil
}

func (h *Server) updateCurrentUser(w http.ResponseWriter, r *http.Request, caller api.ValidUserInfo) (*api.UserInfo, error) {
	target, err := h.parseRequiredUsernameQuery(r)
	if err != nil {
		return nil, err
	}
	var req api.UserUpdateRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}

	updatedUser, err := h.svc.UpdateUser(r.Context(), caller, target, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	return updatedUser, nil
}
