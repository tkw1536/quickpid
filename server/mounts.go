//spellchecker:words server
package server

//spellchecker:words errors http github quickpid
import (
	"errors"
	"fmt"
	"net/http"

	"github.com/tkw1536/quickpid/api"
)

//spellchecker:words nolint err
//nolint:err113 // package-level sentinel used once while creating an annotated error value
var errMountGrantRequired = api.WithErrorCode(errors.New("requires a matching mount grant"), api.Forbidden)

// checkBaseUriGrant wraps a mount handler that needs a validated base URI and a matching mount grant.
// It parses `{baseUri}`, skips the grant check in anonymous mode or for superusers, and otherwise
// requires [service.Service.MountGrantMatches].
func (h *Server) checkBaseUriGrant[T any](
	impl func(http.ResponseWriter, *http.Request, *api.Caller, api.ValidBaseURI) (T, error),
) func(http.ResponseWriter, *http.Request, *api.Caller) (T, error) {
	return func(w http.ResponseWriter, r *http.Request, caller *api.Caller) (T, error) {
		baseURI, err := h.readBaseURIParam(r)
		if err != nil {
			var zero T
			return zero, err
		}
		if !h.svc.AnonymousMode() && caller != nil && !caller.Superuser() {
			ok, err := h.svc.MountGrantMatches(r.Context(), caller.Username(), baseURI)
			if err != nil {
				var zero T
				return zero, fmt.Errorf("failed to check mount grant: %w", err)
			}
			if !ok {
				var zero T
				return zero, errMountGrantRequired
			}
		}
		return impl(w, r, caller, baseURI)
	}
}

func (h *Server) resolveMountByBaseUri(w http.ResponseWriter, r *http.Request, caller *api.Caller) (*api.MountResponse, error) {
	baseURI, err := h.readBaseURIParam(r)
	if err != nil {
		return nil, err
	}
	mount, err := h.svc.GetMount(r.Context(), caller, baseURI)
	if err != nil {
		return nil, fmt.Errorf("failed to get mount: %w", err)
	}
	return mount, nil
}

func (h *Server) resolveResourceByMountAndPID(w http.ResponseWriter, r *http.Request, caller *api.Caller) (api.ResourceGetResult, error) {
	baseURI, err := h.readBaseURIParam(r)
	if err != nil {
		return nil, err
	}
	resourcePID, err := h.readPidParam(r)
	if err != nil {
		return nil, err
	}

	mount, err := h.svc.GetMount(r.Context(), caller, baseURI)
	if err != nil {
		return nil, fmt.Errorf("failed to get mount: %w", err)
	}

	namespace, err := api.NewNamespaceID(mount.Namespace)
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("backend returned invalid namespace id: %w", err), api.DatabaseError)
	}

	resource, err := h.svc.GetResource(r.Context(), caller, namespace, resourcePID, h.shouldRedactResource(r, caller, namespace))
	if err != nil {
		return nil, fmt.Errorf("failed to get resource: %w", err)
	}

	w.Header().Set("Location", h.ops.MountPath+"/resolver/namespaces/"+namespace.String()+"/resources/"+resourcePID.String())
	return resource, nil
}

func (h *Server) listMounts(w http.ResponseWriter, r *http.Request, caller *api.Caller) (*api.PaginatedMountsResponse, error) {
	limit, offset, err := h.parsePagination(r)
	if err != nil {
		return nil, err
	}

	mounts, err := h.svc.ListMounts(r.Context(), caller, api.ListMountsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list mounts: %w", err)
	}
	return mounts, nil
}

func (h *Server) listNamespaceMounts(w http.ResponseWriter, r *http.Request, caller *api.Caller, namespace api.ValidNamespaceID) (*api.PaginatedBaseURIResponse, error) {
	limit, offset, err := h.parsePagination(r)
	if err != nil {
		return nil, err
	}

	mounts, err := h.svc.ListNamespaceMounts(r.Context(), caller, namespace, api.ListNamespaceMountsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespace mounts: %w", err)
	}
	return mounts, nil
}

func (h *Server) upsertNamespaceMount(w http.ResponseWriter, r *http.Request, caller *api.Caller, baseURI api.ValidBaseURI) (*api.MountResponse, error) {
	var req api.MountUpsertRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}

	validReq, err := req.Validate()
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("failed to validate mount upsert request: %w", err), api.InvalidNamespaceID)
	}

	mount, err := h.svc.SetMount(r.Context(), caller, baseURI, validReq)
	if err != nil {
		return nil, fmt.Errorf("failed to set mount: %w", err)
	}
	return mount, nil
}

func (h *Server) deleteNamespaceMount(w http.ResponseWriter, r *http.Request, caller *api.Caller, baseURI api.ValidBaseURI) (struct{}, error) {
	if err := h.svc.DeleteMount(r.Context(), caller, baseURI); err != nil {
		return struct{}{}, fmt.Errorf("failed to delete mount: %w", err)
	}
	return struct{}{}, nil
}

func (h *Server) listOwnMountGrants(w http.ResponseWriter, r *http.Request, caller *api.Caller) (*api.PaginatedMountGrantsResponse, error) {
	limit, offset, err := h.parsePagination(r)
	if err != nil {
		return nil, err
	}

	grants, err := h.svc.ListUserMountGrants(r.Context(), caller, api.ListUserMountGrantsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list mount grants: %w", err)
	}
	return grants, nil
}

func (h *Server) setMountGrant(w http.ResponseWriter, r *http.Request, caller *api.Caller) (*api.MountGrant, error) {
	username, err := h.readUsernameParam(r)
	if err != nil {
		return nil, err
	}

	var req api.MountGrantRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}

	validReq, err := req.Validate()
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("failed to validate mount grant request: %w", err), api.InvalidMountGrantPattern)
	}

	grant, err := h.svc.SetMountGrant(r.Context(), caller, username, validReq)
	if err != nil {
		return nil, fmt.Errorf("failed to set mount grant: %w", err)
	}
	return grant, nil
}

func (h *Server) deleteMountGrant(w http.ResponseWriter, r *http.Request, caller *api.Caller) (struct{}, error) {
	username, err := h.readUsernameParam(r)
	if err != nil {
		return struct{}{}, err
	}

	pattern, err := h.readMountGrantPatternParam(r)
	if err != nil {
		return struct{}{}, err
	}

	if err := h.svc.DeleteMountGrant(r.Context(), caller, username, pattern); err != nil {
		return struct{}{}, fmt.Errorf("failed to delete mount grant: %w", err)
	}
	return struct{}{}, nil
}
