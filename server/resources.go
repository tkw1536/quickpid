//spellchecker:words server
package server

//spellchecker:words errors http strconv github quickpid
import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/tkw1536/quickpid/api"
)

//spellchecker:words lowlevel

func (h *Server) countAllResources(w http.ResponseWriter, r *http.Request, caller *api.Caller) (*api.ResourceCountResponse, error) {
	count, err := h.svc.CountAllResources(r.Context(), caller)
	if err != nil {
		return nil, fmt.Errorf("failed to count all resources: %w", err)
	}
	return count, nil
}

func (h *Server) listResources(w http.ResponseWriter, r *http.Request, caller *api.Caller, namespace api.ValidNamespaceID) (*api.PaginatedResourcesResponse, error) {
	query := r.URL.Query()

	var tag *string
	if query.Has("tag") {
		v := query.Get("tag")
		tag = &v
	}

	var deleted *bool
	if query.Has("deleted") {
		b, err := strconv.ParseBool(query.Get("deleted"))
		if err != nil {
			return nil, api.WithErrorCode(errors.Join(errDeletedInvalid, err), api.InvalidQueryParameter)
		}
		deleted = &b
	}

	limit, offset, err := h.parsePagination(r)
	if err != nil {
		return nil, err
	}

	resources, err := h.svc.ListResources(r.Context(), caller, namespace, api.ListResourcesParams{
		Tag:     tag,
		Deleted: deleted,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}
	return resources, nil
}

func (h *Server) createResource(w http.ResponseWriter, r *http.Request, caller *api.Caller, namespace api.ValidNamespaceID) (*api.ResourceResponse, error) {
	var req api.ResourceCreateRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}

	validReq, err := req.Validate()
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("failed to validate resource create request: %w", err), api.InvalidResourceURL)
	}

	resource, err := h.svc.CreateResource(r.Context(), caller, namespace, validReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}
	return resource, nil
}

func (h *Server) createResourceBatch(w http.ResponseWriter, r *http.Request, caller *api.Caller, namespace api.ValidNamespaceID) ([]api.ResourceResponse, error) {
	var reqs []api.ResourceCreateRequest
	if err := h.decodeJSON(w, r, &reqs); err != nil {
		return nil, err
	}

	validReqs := make([]api.ValidResourceCreateRequest, len(reqs))
	for i := range reqs {
		validReq, err := reqs[i].Validate()
		if err != nil {
			return nil, api.WithErrorCode(fmt.Errorf("failed to validate resource create request: %w", err), api.InvalidResourceURL)
		}
		validReqs[i] = validReq
	}

	resources, err := h.svc.BatchCreateResources(r.Context(), caller, namespace, validReqs)
	if err != nil {
		return nil, fmt.Errorf("failed to batch create resources: %w", err)
	}
	return resources, nil
}

// shouldRedactResource returns a function that can check if the caller should see a redacted resource in the given namespace.
func (h *Server) shouldRedactResource(r *http.Request, caller *api.Caller, namespace api.ValidNamespaceID) func() bool {
	return func() bool {
		return h.lowlevel.CheckNamespaceScope(r, namespace, caller, api.ScopeSeeDeletedResource) != nil
	}
}

func (h *Server) getResource(w http.ResponseWriter, r *http.Request, caller *api.Caller, namespace api.ValidNamespaceID) (api.ResourceGetResult, error) {
	resourcePID, err := h.readPidParam(r)
	if err != nil {
		return nil, err
	}

	resource, err := h.svc.GetResource(r.Context(), caller, namespace, resourcePID, h.shouldRedactResource(r, caller, namespace))
	if err != nil {
		return nil, fmt.Errorf("failed to get resource: %w", err)
	}
	return resource, nil
}

func (h *Server) updateResource(w http.ResponseWriter, r *http.Request, caller *api.Caller, namespace api.ValidNamespaceID) (*api.ResourceResponse, error) {
	var req api.ResourceUpdateRequest
	if err := h.decodeJSON(w, r, &req); err != nil {
		return nil, err
	}

	validReq, err := req.Validate()
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("failed to validate resource update request: %w", err), api.InvalidResourceURL)
	}

	resourcePID, err := h.readPidParam(r)
	if err != nil {
		return nil, err
	}

	resource, err := h.svc.UpdateResource(r.Context(), caller, namespace, resourcePID, validReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update resource: %w", err)
	}
	return resource, nil
}
