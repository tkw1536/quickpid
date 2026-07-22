//spellchecker:words server
package server

//spellchecker:words http github quickpid
import (
	"fmt"
	"net/http"

	"github.com/tkw1536/quickpid/api"
)

func (h *Server) getMount(w http.ResponseWriter, r *http.Request) (*api.MountResponse, error) {
	baseURI, err := h.getBaseURI(r)
	if err != nil {
		return nil, err
	}
	mount, err := h.svc.GetMount(r.Context(), baseURI)
	if err != nil {
		return nil, fmt.Errorf("svc.GetMount: %w", err)
	}
	return mount, nil
}

func (h *Server) listMounts(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (*api.PaginatedMountsResponse, error) {
	limit, offset, err := h.parsePagination(r)
	if err != nil {
		return nil, err
	}

	mounts, err := h.svc.ListMounts(r.Context(), user, api.ListMountsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("svc.ListMounts: %w", err)
	}
	return mounts, nil
}

func (h *Server) listNamespaceMounts(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (*api.PaginatedBaseURIResponse, error) {
	namespace, err := h.getNamespace(r)
	if err != nil {
		return nil, err
	}
	limit, offset, err := h.parsePagination(r)
	if err != nil {
		return nil, err
	}

	mounts, err := h.svc.ListNamespaceMounts(r.Context(), user, namespace, api.ListNamespaceMountsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("svc.ListNamespaceMounts: %w", err)
	}
	return mounts, nil
}

func (h *Server) setMount(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (*api.MountResponse, error) {
	baseURI, err := h.getBaseURI(r)
	if err != nil {
		return nil, err
	}

	var req api.MountUpsertRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}

	validReq, err := req.Validate()
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("failed to validate mount upsert request: %w", err), api.InvalidNamespaceID)
	}

	mount, err := h.svc.SetMount(r.Context(), user, baseURI, validReq)
	if err != nil {
		return nil, fmt.Errorf("svc.SetMount: %w", err)
	}
	return mount, nil
}

func (h *Server) deleteMount(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (struct{}, error) {
	baseURI, err := h.getBaseURI(r)
	if err != nil {
		return struct{}{}, err
	}
	if err := h.svc.DeleteMount(r.Context(), user, baseURI); err != nil {
		return struct{}{}, fmt.Errorf("svc.DeleteMount: %w", err)
	}
	return struct{}{}, nil
}
