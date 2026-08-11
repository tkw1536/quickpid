//spellchecker:words server
package server

//spellchecker:words http github quickpid
import (
	"fmt"
	"net/http"

	"github.com/tkw1536/quickpid/api"
)

func (h *Server) resolveMountByBaseUri(w http.ResponseWriter, r *http.Request) (*api.MountResponse, error) {
	baseURI, err := h.readBaseURIParam(r)
	if err != nil {
		return nil, err
	}
	mount, err := h.svc.GetMount(r.Context(), baseURI)
	if err != nil {
		return nil, fmt.Errorf("failed to get mount: %w", err)
	}
	return mount, nil
}

func (h *Server) resolveResourceByMountAndPID(w http.ResponseWriter, r *http.Request, caller *api.ValidUserInfo) (api.ResourceGetResult, error) {
	baseURI, err := h.readBaseURIParam(r)
	if err != nil {
		return nil, err
	}
	resourcePID, err := h.readPidParam(r)
	if err != nil {
		return nil, err
	}

	mount, err := h.svc.GetMount(r.Context(), baseURI)
	if err != nil {
		return nil, fmt.Errorf("failed to get mount: %w", err)
	}

	namespace, err := api.NewNamespaceID(mount.Namespace)
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("backend returned invalid namespace id: %w", err), api.DatabaseError)
	}

	resource, err := h.svc.GetResource(r.Context(), caller, namespace, resourcePID)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource: %w", err)
	}

	w.Header().Set("Location", h.ops.MountPath+"/resolver/namespaces/"+namespace.String()+"/resources/"+resourcePID.String())
	return resource, nil
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
		return nil, fmt.Errorf("failed to list mounts: %w", err)
	}
	return mounts, nil
}

func (h *Server) listNamespaceMounts(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (*api.PaginatedBaseURIResponse, error) {
	namespace, err := h.readNamespaceParam(r)
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
		return nil, fmt.Errorf("failed to list namespace mounts: %w", err)
	}
	return mounts, nil
}

func (h *Server) upsertNamespaceMount(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (*api.MountResponse, error) {
	baseURI, err := h.readBaseURIParam(r)
	if err != nil {
		return nil, err
	}

	var req api.MountUpsertRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}

	validReq, err := req.Validate()
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("failed to validate mount upsert request: %w", err), api.InvalidNamespaceID)
	}

	mount, err := h.svc.SetMount(r.Context(), user, baseURI, validReq)
	if err != nil {
		return nil, fmt.Errorf("failed to set mount: %w", err)
	}
	return mount, nil
}

func (h *Server) deleteNamespaceMount(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (struct{}, error) {
	baseURI, err := h.readBaseURIParam(r)
	if err != nil {
		return struct{}{}, err
	}
	if err := h.svc.DeleteMount(r.Context(), user, baseURI); err != nil {
		return struct{}{}, fmt.Errorf("failed to delete mount: %w", err)
	}
	return struct{}{}, nil
}
