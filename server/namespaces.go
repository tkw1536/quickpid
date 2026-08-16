//spellchecker:words server
package server

//spellchecker:words context http github quickpid
import (
	"context"
	"fmt"
	"net/http"

	"github.com/tkw1536/quickpid/api"
)

//spellchecker:words lowlevel

func (h *Server) listNamespaces(w http.ResponseWriter, r *http.Request, caller *api.Caller) (*api.PaginatedNamespacesResponse, error) {
	limit, offset, err := h.parsePagination(r)
	if err != nil {
		return nil, err
	}

	query := r.URL.Query()
	var tag *string
	if query.Has("tag") {
		v := query.Get("tag")
		tag = &v
	}

	namespaces, err := h.svc.ListNamespaces(r.Context(), caller, api.ListNamespacesParams{
		Tag:    tag,
		Limit:  limit,
		Offset: offset,
	}, func(ctx context.Context, namespace api.ValidNamespaceID) bool {
		return h.lowlevel.CheckNamespaceScope(r, namespace, caller, api.ScopeGetNamespace) == nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}
	return namespaces, nil
}

func (h *Server) getNamespace(w http.ResponseWriter, r *http.Request, caller *api.Caller, namespace api.ValidNamespaceID) (*api.NamespaceResponse, error) {
	res, err := h.svc.GetNamespace(r.Context(), caller, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace: %w", err)
	}
	return res, nil
}

func (h *Server) updateNamespace(w http.ResponseWriter, r *http.Request, caller *api.Caller, namespace api.ValidNamespaceID) (*api.NamespaceResponse, error) {
	var req api.NamespaceUpdateRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}

	validReq, err := req.Validate()
	if err != nil {
		return nil, fmt.Errorf("failed to validate namespace update request: %w", err)
	}

	res, err := h.svc.UpdateNamespace(r.Context(), caller, namespace, validReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update namespace: %w", err)
	}
	return res, nil
}

func (h *Server) createNamespace(w http.ResponseWriter, r *http.Request, caller *api.Caller) (*api.NamespaceResponse, error) {
	var req api.NamespaceCreateRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}
	namespace, err := h.svc.CreateNamespace(r.Context(), caller, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create namespace: %w", err)
	}
	return namespace, nil
}
