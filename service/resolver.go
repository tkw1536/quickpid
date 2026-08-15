//spellchecker:words service
package service

//spellchecker:words context errors github quickpid backend scopes
import (
	"context"
	"errors"
	"fmt"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/pid"
	"github.com/tkw1536/quickpid/scopes"
)

var errExistingUserNotFound = errors.New("existing user not found")

// ListNamespaces lists namespaces.
//
// It can return the following errors:
//
// - [api.Unauthorized]
// - [api.DatabaseError].
func (s *Service) ListNamespaces(ctx context.Context, caller *api.Caller, params api.ListNamespacesParams) (*api.PaginatedNamespacesResponse, error) {
	if err := s.checkUserScope(caller, scopes.ScopeListNamespaces); err != nil {
		return nil, fmt.Errorf("ListNamespaces() check failed: %w", err)
	}

	var user *api.ValidUsername
	if caller != nil && !caller.Superuser() {
		user = new(caller.Username())
	}
	// TODO: Filter this caller site ...
	out, err := s.store.ListNamespaces(ctx, user, params)

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

// GetNamespace returns one namespace by id.
//
// It can return the following errors:
//
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.NamespaceNotFound]
// - [api.DatabaseError].
func (s *Service) GetNamespace(ctx context.Context, caller *api.Caller, namespace api.ValidNamespaceID) (*api.NamespaceResponse, error) {
	if err := s.checkNamespaceScope(ctx, namespace, caller, scopes.ScopeReadMetadata); err != nil {
		return nil, fmt.Errorf("ReadMetadata() check failed: %w", err)
	}

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
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.NamespaceNotFound]
// - [api.DatabaseError].
func (s *Service) UpdateNamespace(ctx context.Context, caller *api.Caller, namespace api.ValidNamespaceID, req api.ValidNamespaceUpdateRequest) (*api.NamespaceResponse, error) {
	if err := s.checkNamespaceScope(ctx, namespace, caller, scopes.ScopePatchMetadata); err != nil {
		return nil, fmt.Errorf("PatchMetadata() check failed: %w", err)
	}

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
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.DatabaseError].
func (s *Service) CountAllResources(ctx context.Context, caller *api.Caller) (*api.ResourceCountResponse, error) {
	if err := s.checkUserScope(caller, scopes.ScopeCountAllResources); err != nil {
		return nil, fmt.Errorf("CountAllResources() check failed: %w", err)
	}

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
// - [api.Unauthorized]
// - [api.DatabaseError]
// - [api.BadIDGeneration]
// - [api.InsufficientEntropy].
func (s *Service) CreateNamespace(ctx context.Context, caller *api.Caller, req api.NamespaceCreateRequest) (*api.NamespaceResponse, error) {
	if err := s.checkUserScope(caller, scopes.ScopeCreateNamespace); err != nil {
		return nil, fmt.Errorf("CreateNamespace() check failed: %w", err)
	}

	s.mu.RLock()
	maxAttempts := s.opts.Limits.MaxNamespaceIDAttempts
	s.mu.RUnlock()

	var owner *api.ValidUsername
	if !s.AnonymousMode() {
		if err := requireAuthenticated(caller); err != nil {
			return nil, err
		}
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
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.NamespaceNotFound]
// - [api.DatabaseError].
func (s *Service) ListResources(ctx context.Context, caller *api.Caller, namespace api.ValidNamespaceID, params api.ListResourcesParams) (*api.PaginatedResourcesResponse, error) {
	if err := s.checkNamespaceScope(ctx, namespace, caller, scopes.ScopeListResources); err != nil {
		return nil, fmt.Errorf("ListResources() check failed: %w", err)
	}

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
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.NamespaceNotFound]
// - [api.DatabaseError]
// - [api.BadIDGeneration]
// - [api.InsufficientEntropy].
func (s *Service) CreateResource(ctx context.Context, caller *api.Caller, namespace api.ValidNamespaceID, req api.ValidResourceCreateRequest) (*api.ResourceResponse, error) {
	if err := s.checkNamespaceScope(ctx, namespace, caller, scopes.ScopeCreateResource); err != nil {
		return nil, fmt.Errorf("CreateResource() check failed: %w", err)
	}

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
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.NamespaceNotFound]
// - [api.DatabaseError]
// - [api.BadIDGeneration]
// - [api.InsufficientEntropy].
func (s *Service) BatchCreateResources(ctx context.Context, caller *api.Caller, namespace api.ValidNamespaceID, reqs []api.ValidResourceCreateRequest) ([]api.ResourceResponse, error) {
	if err := s.checkNamespaceScope(ctx, namespace, caller, scopes.ScopeCreateResource); err != nil {
		return nil, fmt.Errorf("BatchCreateResources() check failed: %w", err)
	}

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
// When the resource is deleted and the caller is not allowed to see the full
// object, it returns a [api.RedactedResourceResponse] (HTTP 410) instead of an error.
func (s *Service) GetResource(ctx context.Context, caller *api.Caller, namespace api.ValidNamespaceID, resourcePID api.ValidPID) (api.ResourceGetResult, error) {
	if err := s.checkNamespaceScope(ctx, namespace, caller, scopes.ScopeGetResource); err != nil {
		return nil, fmt.Errorf("GetResource() check failed: %w", err)
	}

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

	if out.Deleted {
		if err := s.checkNamespaceScope(ctx, namespace, caller, scopes.ScopeSeeDeletedResource); err != nil {
			return out.Redact(), nil
		}
	}
	return out, nil
}

// UpdateResource updates a resource by namespace and pid.
//
// It can return the following errors:
//
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.DatabaseError]
// - [api.NamespaceNotFound]
// - [api.ResourceNotFound].
func (s *Service) UpdateResource(ctx context.Context, caller *api.Caller, namespace api.ValidNamespaceID, resourcePID api.ValidPID, req api.ValidResourceUpdateRequest) (*api.ResourceResponse, error) {
	if err := s.checkNamespaceScope(ctx, namespace, caller, scopes.ScopeUpdateResource); err != nil {
		return nil, fmt.Errorf("UpdateResource() check failed: %w", err)
	}

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
