//spellchecker:words server
package server

//spellchecker:words http github quickpid
import (
	"fmt"
	"net/http"

	"github.com/tkw1536/quickpid/api"
)

func (h *Server) getUserInfo(w http.ResponseWriter, r *http.Request, caller *api.Caller) (*api.UserInfo, error) {
	info := caller.PlainInfo()
	return &info, nil
}

func (h *Server) createUser(w http.ResponseWriter, r *http.Request, caller *api.Caller) (*api.UserInfo, error) {
	var req api.UserCreateRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}

	validReq, err := req.Validate()
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("failed to validate user create request: %w", err), api.InvalidUsername)
	}

	createdUser, err := h.svc.CreateUser(r.Context(), *caller, validReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return createdUser, nil
}

func (h *Server) deleteUser(w http.ResponseWriter, r *http.Request, caller *api.Caller) (struct{}, error) {
	target, err := h.readRequiredUsernameFromQuery(r)
	if err != nil {
		return struct{}{}, err
	}

	if err := h.svc.DeleteUser(r.Context(), *caller, target); err != nil {
		return struct{}{}, fmt.Errorf("failed to delete user: %w", err)
	}
	return struct{}{}, nil
}

func (h *Server) listUsers(w http.ResponseWriter, r *http.Request, caller *api.Caller) (*api.PaginatedUsersResponse, error) {
	superuser, err := h.readOptionalBooleanFromQuery(r, "superuser")
	if err != nil {
		return nil, err
	}
	password, err := h.readOptionalBooleanFromQuery(r, "password")
	if err != nil {
		return nil, err
	}
	limit, offset, err := h.parsePagination(r)
	if err != nil {
		return nil, err
	}

	users, err := h.svc.ListUsers(r.Context(), *caller, api.ListUsersParams{
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

func (h *Server) autocompleteUsers(w http.ResponseWriter, r *http.Request, caller *api.Caller) ([]string, error) {
	query, err := h.readRequiredAutocompleteFromQuery(r)
	if err != nil {
		return nil, err
	}
	usernames, err := h.svc.AutocompleteUsers(r.Context(), *caller, query)
	if err != nil {
		return nil, fmt.Errorf("failed to autocomplete users: %w", err)
	}
	return usernames, nil
}

func (h *Server) updateUser(w http.ResponseWriter, r *http.Request, caller *api.Caller) (*api.UserInfo, error) {
	target, err := h.readRequiredUsernameFromQuery(r)
	if err != nil {
		return nil, err
	}
	var req api.UserUpdateRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}

	updatedUser, err := h.svc.UpdateUser(r.Context(), *caller, target, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	return updatedUser, nil
}
