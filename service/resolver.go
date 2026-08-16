//spellchecker:words service
package service

//spellchecker:words context errors github quickpid backend internal filter
import (
	"context"
	"errors"
	"fmt"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/internal/filter"
	"github.com/tkw1536/quickpid/pid"
)

var errExistingUserNotFound = errors.New("existing user not found")

// ListNamespaces lists namespaces.
//
// It can return the following errors:
//
// - [api.DatabaseError].
func (s *Service) ListNamespaces(ctx context.Context, caller *api.Caller, params api.ListNamespacesParams, checkAccess func(ctx context.Context, namespace api.ValidNamespaceID) bool) (*api.PaginatedNamespacesResponse, error) {
	var callerToFilterBy *api.Caller
	if caller != nil && !caller.Superuser() {
		callerToFilterBy = caller
	}
	out, err := s.listVisibleNamespaces(ctx, callerToFilterBy, params, checkAccess)

	if errors.Is(err, backend.ErrUserNotFound) {
		// This SHOULD NEVER happen as we received the username from a user object.
		// But a race condition between a concurrent delete user call or data corruption might trigger this.
		// So we consider this a database error.
		return nil, api.WithErrorCode(fmt.Errorf("%w: %w", errExistingUserNotFound, err), api.DatabaseError)
	}
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to list namespaces: %w", err), api.DatabaseError)
	}
	return out, nil
}

func (s *Service) listVisibleNamespaces(ctx context.Context, caller *api.Caller, params api.ListNamespacesParams, checkAccess func(ctx context.Context, namespace api.ValidNamespaceID) bool) (*api.PaginatedNamespacesResponse, error) {
	// No filtering to apply: just directly call the backend.
	if caller == nil {
		namespaces, err := s.store.ListNamespaces(ctx, params)
		if err != nil {
			return nil, api.WithErrorCode(fmt.Errorf("backend failed to list namespaces: %w", err), api.DatabaseError)
		}
		return namespaces, nil
	}

	// Determine the effective page limit to call the backend with.
	options := s.Options().Limits
	pageLimit := options.DefaultPageLimit
	if options.MaxPageLimit > 0 && pageLimit > options.MaxPageLimit {
		pageLimit = options.MaxPageLimit
	}

	filtered, err := filter.Filter(ctx, func(ctx context.Context, limit, offset int) ([]api.NamespaceResponse, error) {
		params := params
		params.Limit = limit
		params.Offset = offset

		namespaces, err := s.store.ListNamespaces(ctx, params)
		if err != nil {
			return nil, api.WithErrorCode(fmt.Errorf("backend failed to list namespaces: %w", err), api.DatabaseError)
		}
		return namespaces.Items, nil
	}, func(ctx context.Context, namespace api.NamespaceResponse) (bool, error) {
		name, err := api.NewNamespaceID(namespace.ID)
		if err != nil {
			return false, api.WithErrorCode(fmt.Errorf("backend returned invalid namespace id: %w", err), api.DatabaseError)
		}
		return checkAccess(ctx, name), nil
	}, pageLimit, params.Limit, params.Offset)

	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("filter returned error: %w", err), api.DatabaseError)
	}
	return &api.PaginatedNamespacesResponse{
		Items:  filtered.Items,
		Total:  filtered.Total,
		Offset: filtered.Offset,
	}, nil
}

// GetNamespace returns one namespace by id.
//
// It can return the following errors:
//
// - [api.NamespaceNotFound]
// - [api.DatabaseError].
func (s *Service) GetNamespace(ctx context.Context, caller *api.Caller, namespace api.ValidNamespaceID) (*api.NamespaceResponse, error) {
	out, err := s.store.GetNamespace(ctx, namespace)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.WithErrorCode(fmt.Errorf("namespace not found: %w", err), api.NamespaceNotFound)
	}
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to get namespace: %w", err), api.DatabaseError)
	}
	return out, nil
}

// UpdateNamespace updates a namespace by id.
//
// It can return the following errors:
//
// - [api.NamespaceNotFound]
// - [api.DatabaseError].
func (s *Service) UpdateNamespace(ctx context.Context, caller *api.Caller, namespace api.ValidNamespaceID, req api.ValidNamespaceUpdateRequest) (*api.NamespaceResponse, error) {
	out, err := s.store.UpdateNamespace(ctx, namespace, req, s.runtime.Now)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.WithErrorCode(fmt.Errorf("namespace not found: %w", err), api.NamespaceNotFound)
	}
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to update namespace: %w", err), api.DatabaseError)
	}
	return out, nil
}

// CountAllResources returns the total number of resources across all namespaces.
//
// It can return the following errors:
//
// - [api.DatabaseError].
func (s *Service) CountAllResources(ctx context.Context, caller *api.Caller) (*api.ResourceCountResponse, error) {
	n, err := s.store.CountAllResources(ctx)
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to count all resources: %w", err), api.DatabaseError)
	}
	return &api.ResourceCountResponse{Total: int(n)}, nil
}

// CreateNamespace creates a new namespace owned by the authenticated caller.
//
// It can return the following errors:
//
// - [api.DatabaseError]
// - [api.BadIDGeneration]
// - [api.InsufficientEntropy].
func (s *Service) CreateNamespace(ctx context.Context, caller *api.Caller, req api.NamespaceCreateRequest) (*api.NamespaceResponse, error) {
	s.mu.RLock()
	maxAttempts := s.opts.Limits.MaxNamespaceIDAttempts
	s.mu.RUnlock()

	var owner *api.ValidUsername
	if caller != nil {
		owner = new(caller.Username())
	}
	for range maxAttempts {
		name, err := s.runtime.NewNamespaceID()
		if err != nil {
			return nil, api.WithErrorCode(fmt.Errorf("runtime failed to generate namespace ID: %w", err), api.BadIDGeneration)
		}
		namespace, err := api.NewNamespaceID(name)
		if err != nil {
			return nil, api.WithErrorCode(fmt.Errorf("%w: %q is not a valid namespace id", errBadNamespaceID, name), api.BadIDGeneration)
		}
		out, err := s.store.CreateNamespace(ctx, namespace, req, owner, s.runtime.Now)
		if err == nil {
			return out, nil
		}
		if errors.Is(err, backend.ErrUserNotFound) {
			// This SHOULD NEVER happen as we received the username from a user object.
			// But a race condition between a concurrent delete user call or data corruption might trigger this.
			// So we consider this a database error.
			return nil, api.WithErrorCode(fmt.Errorf("%w: %w", errExistingUserNotFound, err), api.DatabaseError)
		}
		if !errors.Is(err, backend.ErrDuplicateNamespaceID) {
			return nil, api.WithErrorCode(fmt.Errorf("backend failed to create namespace: %w", err), api.DatabaseError)
		}
	}
	return nil, api.WithErrorCode(fmt.Errorf("%w: gave up namespace id generation after %d attempts", errInsufficientEntropy, maxAttempts), api.InsufficientEntropy)
}

// ListResources lists resources in a namespace.
//
// It can return the following errors:
//
// - [api.NamespaceNotFound]
// - [api.DatabaseError].
func (s *Service) ListResources(ctx context.Context, caller *api.Caller, namespace api.ValidNamespaceID, params api.ListResourcesParams) (*api.PaginatedResourcesResponse, error) {
	out, err := s.store.ListResources(ctx, namespace, params)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.WithErrorCode(fmt.Errorf("namespace not found: %w", err), api.NamespaceNotFound)
	}
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to list resources: %w", err), api.DatabaseError)
	}
	return out, nil
}

// CreateResource creates a single resource.
//
// It can return the following errors:
//
// - [api.NamespaceNotFound]
// - [api.DatabaseError]
// - [api.BadIDGeneration]
// - [api.InsufficientEntropy].
func (s *Service) CreateResource(ctx context.Context, caller *api.Caller, namespace api.ValidNamespaceID, req api.ValidResourceCreateRequest) (*api.ResourceResponse, error) {
	ns, err := s.store.GetNamespace(ctx, namespace)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.WithErrorCode(fmt.Errorf("namespace not found: %w", err), api.NamespaceNotFound)
	}
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to get namespace: %w", err), api.DatabaseError)
	}

	out, err := s.allocatePID(ns.PIDFormat, func(pid api.ValidPID) (*api.ResourceResponse, error) {
		return s.store.CreateResource(ctx, namespace, pid, req, s.runtime.Now)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

var errLimitExceeded = errors.New("batch create limit exceeded")

// BatchCreateResources creates multiple resources in a namespace.
//
// It can return the following errors:
//
// - [api.ItemLimitExceeded]
// - [api.NamespaceNotFound]
// - [api.DatabaseError]
// - [api.BadIDGeneration]
// - [api.InsufficientEntropy].
func (s *Service) BatchCreateResources(ctx context.Context, caller *api.Caller, namespace api.ValidNamespaceID, reqs []api.ValidResourceCreateRequest) ([]api.ResourceResponse, error) {
	s.mu.RLock()
	maxBatch := s.opts.Limits.MaxBatchItems
	s.mu.RUnlock()

	if maxBatch > 0 && len(reqs) > maxBatch {
		return nil, api.WithErrorCode(fmt.Errorf("%w: %d > %d", errLimitExceeded, len(reqs), maxBatch), api.ItemLimitExceeded)
	}

	ns, err := s.store.GetNamespace(ctx, namespace)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.WithErrorCode(fmt.Errorf("namespace not found: %w", err), api.NamespaceNotFound)
	}
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to get namespace: %w", err), api.DatabaseError)
	}

	out, err := s.allocatePIDs(ns.PIDFormat, len(reqs), func(pids []api.ValidPID) ([]api.ResourceResponse, error) {
		return s.store.BatchCreateResources(ctx, namespace, pids, reqs, s.runtime.Now)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetResource gets a resource by namespace and pid.
//
// It can return the following errors:
//
// - [api.Unauthorized]
// - [api.NamespaceNotFound]
// - [api.ResourceNotFound]
// - [api.DatabaseError].
//
// When redact is true, and the resource is deleted, the caller is not allowed to see the full
// object, and this function returns a [api.RedactedResourceResponse] (HTTP 410) instead of an error.
func (s *Service) GetResource(ctx context.Context, caller *api.Caller, namespace api.ValidNamespaceID, resourcePID api.ValidPID, shouldRedact func() bool) (api.ResourceGetResult, error) {
	out, err := s.store.GetResource(ctx, namespace, resourcePID)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.WithErrorCode(fmt.Errorf("namespace not found: %w", err), api.NamespaceNotFound)
	}
	if errors.Is(err, backend.ErrResourceNotFound) {
		return nil, api.WithErrorCode(fmt.Errorf("resource not found: %w", err), api.ResourceNotFound)
	}
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to get resource: %w", err), api.DatabaseError)
	}

	if out.Deleted && shouldRedact() {
		return out.Redact(), nil
	}
	return out, nil
}

// UpdateResource updates a resource by namespace and pid.
//
// It can return the following errors:
//
// - [api.DatabaseError]
// - [api.NamespaceNotFound]
// - [api.ResourceNotFound].
func (s *Service) UpdateResource(ctx context.Context, caller *api.Caller, namespace api.ValidNamespaceID, resourcePID api.ValidPID, req api.ValidResourceUpdateRequest) (*api.ResourceResponse, error) {
	out, err := s.store.UpdateResource(ctx, namespace, resourcePID, req, s.runtime.Now)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.WithErrorCode(fmt.Errorf("namespace not found: %w", err), api.NamespaceNotFound)
	}
	if errors.Is(err, backend.ErrResourceNotFound) {
		return nil, api.WithErrorCode(fmt.Errorf("resource not found: %w", err), api.ResourceNotFound)
	}
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to update resource: %w", err), api.DatabaseError)
	}
	return out, nil
}

// allocatePIDs allocates n unique pids in the given namespace.
//
// It can return the following errors:
//
// - [api.BadIDGeneration]
// - [api.DatabaseError]
// - [api.InsufficientEntropy].
func (s *Service) allocatePIDs(format pid.Format, n int, insert func([]api.ValidPID) ([]api.ResourceResponse, error)) ([]api.ResourceResponse, error) {
	if n == 0 {
		return []api.ResourceResponse{}, nil
	}

	s.mu.RLock()
	maxAttempts := s.opts.Limits.MaxPIDAttempts
	s.mu.RUnlock()

	for range maxAttempts {
		pids := make([]api.ValidPID, n)
		seen := make(map[string]struct{}, n)
		for i := range n {
			// Ensure uniqueness within this batch.
			for range maxAttempts {
				candidate, err := s.runtime.NewPID(format)
				if err != nil {
					return nil, api.WithErrorCode(fmt.Errorf("runtime failed to generate PID: %w", err), api.BadIDGeneration)
				}
				if _, exists := seen[candidate]; exists {
					continue
				}
				seen[candidate] = struct{}{}
				pids[i], err = api.NewPID(candidate)
				if err != nil {
					return nil, api.WithErrorCode(fmt.Errorf("runtime returned invalid PID: %w", err), api.BadIDGeneration)
				}
				break
			}

			if !format.IsValid(pids[i].String()) {
				return nil, api.WithErrorCode(fmt.Errorf("%w: %q is not a valid pid", errBadPID, pids[i]), api.BadIDGeneration)
			}
		}

		out, err := insert(pids)
		if err == nil {
			return out, nil
		}
		if !errors.Is(err, backend.ErrPIDAllocationFailed) {
			return nil, api.WithErrorCode(err, api.DatabaseError)
		}
	}
	return nil, api.WithErrorCode(fmt.Errorf("%w: gave up pid generation after %d attempts", errInsufficientEntropy, maxAttempts), api.InsufficientEntropy)
}

// allocatePID is like allocatePIDs but for a single PID.
func (s *Service) allocatePID(format pid.Format, insert func(api.ValidPID) (*api.ResourceResponse, error)) (*api.ResourceResponse, error) {
	pids, err := s.allocatePIDs(format, 1, func(pids []api.ValidPID) ([]api.ResourceResponse, error) {
		res, err := insert(pids[0])
		if err != nil {
			return nil, err
		}
		return []api.ResourceResponse{*res}, nil
	})
	if err != nil {
		return nil, err
	}
	p := pids[0]
	return &p, nil
}
