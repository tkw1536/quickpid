//spellchecker:words server
package server

//spellchecker:words http github quickpid
import (
	"fmt"
	"net/http"

	"github.com/tkw1536/quickpid/api"
)

func (h *Server) setUserPassword(w http.ResponseWriter, r *http.Request, caller *api.Caller) (*api.SetPasswordResponse, error) {
	var req api.SetPasswordRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}

	validReq, err := req.Validate()
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("failed to validate set password request: %w", err), api.InvalidPassword)
	}

	response, err := h.svc.SetUserPassword(r.Context(), *caller, validReq)
	if err != nil {
		return nil, fmt.Errorf("failed to set user password: %w", err)
	}
	return response, nil
}

func (h *Server) listUserKeys(w http.ResponseWriter, r *http.Request, caller *api.Caller) (*api.PaginatedAPIKeysResponse, error) {
	limit, offset, err := h.parsePagination(r)
	if err != nil {
		return nil, err
	}

	keys, err := h.svc.ListKeys(r.Context(), *caller, api.ListKeysParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}
	return keys, nil
}

func (h *Server) issueUserKey(w http.ResponseWriter, r *http.Request, caller *api.Caller) (*api.IssueKeyResponse, error) {
	var req api.KeyIssueRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}

	response, err := h.svc.IssueKey(r.Context(), *caller, req)
	if err != nil {
		return nil, fmt.Errorf("failed to issue key: %w", err)
	}
	return response, nil
}

func (h *Server) revokeUserKey(w http.ResponseWriter, r *http.Request, caller *api.Caller) (struct{}, error) {
	var req api.KeyRevokeRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return struct{}{}, err
	}

	if err := h.svc.RevokeKey(r.Context(), *caller, req); err != nil {
		return struct{}{}, fmt.Errorf("failed to revoke key: %w", err)
	}
	return struct{}{}, nil
}
