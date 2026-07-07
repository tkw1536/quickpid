//spellchecker:words server
package server

//spellchecker:words http github bicpid
import (
	"net/http"

	"github.com/tkw1536/bicpid/api"
)

func (h *Server) getCurrentUserHTTP(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (*api.UserInfo, api.Error, error) {
	return user, "", nil
}

func (h *Server) createUser(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (*api.UserInfo, api.Error, error) {
	var req api.UserCreateRequest
	if specError, err := h.decodeJSON(w, r, &req); err != nil {
		return nil, specError, err
	}

	return h.svc.CreateUser(r.Context(), user, req)
}

func (h *Server) listUsers(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (*api.PaginatedUsersResponse, api.Error, error) {
	limit, offset, specError, err := h.parsePagination(r)
	if err != nil {
		return nil, specError, err
	}

	return h.svc.ListUsers(r.Context(), user, api.ListUsersParams{
		Limit:  limit,
		Offset: offset,
	})
}

func (h *Server) listKeys(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (*api.PaginatedAPIKeysResponse, api.Error, error) {
	limit, offset, specError, err := h.parsePagination(r)
	if err != nil {
		return nil, specError, err
	}

	return h.svc.ListKeys(r.Context(), user, api.ListKeysParams{
		Limit:  limit,
		Offset: offset,
	})
}

func (h *Server) issueKey(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (*api.IssueKeyResponse, api.Error, error) {
	var req api.IssueKeyRequest
	if specError, err := h.decodeJSON(w, r, &req); err != nil {
		return nil, specError, err
	}

	return h.svc.IssueKey(r.Context(), user, req)
}

func (h *Server) revokeKey(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (*api.RevokeKeyResponse, api.Error, error) {
	var req api.RevokeKeyRequest
	if specError, err := h.decodeJSON(w, r, &req); err != nil {
		return nil, specError, err
	}

	return h.svc.RevokeKey(r.Context(), user, req)
}

func (h *Server) updateCurrentUser(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (*api.UserInfo, api.Error, error) {
	var req api.UpdateUserRequest
	if specError, err := h.decodeJSON(w, r, &req); err != nil {
		return nil, specError, err
	}

	return h.svc.UpdateUser(r.Context(), user, req)
}
