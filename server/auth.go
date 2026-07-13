//spellchecker:words server
package server

//spellchecker:words http github bicpid
import (
	"net/http"

	"github.com/tkw1536/bicpid/api"
)

func (h *Server) getCurrentUserHTTP(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (*api.UserInfo, api.Error, error) {
	target, specError, err := h.parseOptionalUsernameQuery(r)
	if err != nil {
		return nil, specError, err
	}
	return h.svc.GetUserAccount(r.Context(), user, target)
}

func (h *Server) createUser(w http.ResponseWriter, r *http.Request, user *api.UserInfo) (*api.UserInfo, api.Error, error) {
	var req api.UserCreateRequest
	if specError, err := h.decodeJSON(w, r, &req); err != nil {
		return nil, specError, err
	}

	return h.svc.CreateUser(r.Context(), user, req)
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

	return h.svc.ListUsers(r.Context(), user, api.ListUsersParams{
		Superuser: superuser,
		Limit:     limit,
		Offset:    offset,
	})
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

	return h.svc.ListKeys(r.Context(), user, target, api.ListKeysParams{
		Limit:  limit,
		Offset: offset,
	})
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

	return h.svc.IssueKey(r.Context(), user, target, req)
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

	return h.svc.RevokeKey(r.Context(), user, target, req)
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

	return h.svc.UpdateUser(r.Context(), user, target, req)
}
