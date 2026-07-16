//spellchecker:words service
package service

//spellchecker:words context errors github quickpid backend
import (
	"context"
	"errors"
	"fmt"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/pid"
)

var errExistingUserNotFound = errors.New("existing user not found")

// ListNamespaces lists namespaces.
//
// It can return the following errors:
//
// - [api.Unauthorized]
// - [api.DatabaseError].
func (s *Service) ListNamespaces(ctx context.Context, caller *api.UserInfo, params api.ListNamespacesParams) (*api.PaginatedNamespacesResponse, error) {
	// in authenticated mode, require the caller to be authenticated.
	if !s.Options().Anonymous {
		if err := s.requireAuthenticated(caller); err != nil {
			return nil, err
		}
	}

	var user *string
	if caller != nil && !caller.Superuser {
		user = new(caller.Username)
	}
	out, err := s.store.ListNamespaces(ctx, user, params)
	if errors.Is(err, backend.ErrUserNotFound) {
		// This SHOULD NEVER happen as we received the username from a user object.
		// But a race condition between a concurrent delete user call or data corruption might trigger this.
		// So we consider this a database error.
		return nil, api.WithErrorString(fmt.Errorf("%w: %w", errExistingUserNotFound, err), api.DatabaseError)
	}
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("store.ListNamespaces: %w", err), api.DatabaseError)
	}
	return out, nil
}

// GetNamespace returns one namespace by id.
//
// It can return the following errors:
//
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.InvalidNamespaceID]
// - [api.NamespaceNotFound]
// - [api.DatabaseError].
func (s *Service) GetNamespace(ctx context.Context, caller *api.UserInfo, namespace string) (*api.NamespaceResponse, error) {
	if s.Options().Anonymous {
		if err := ValidateNamespaceID(namespace); err != nil {
			return nil, api.WithErrorString(fmt.Errorf("ValidateNamespaceID: %w", err), api.InvalidNamespaceID)
		}
	} else {
		if err := s.requireNamespaceCapability(ctx, caller, namespace, canReadNamespaceMetadata); err != nil {
			return nil, err
		}
	}
	out, err := s.store.GetNamespace(ctx, namespace)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.WithErrorString(fmt.Errorf("store.GetNamespace: %w", err), api.NamespaceNotFound)
	}
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("store.GetNamespace: %w", err), api.DatabaseError)
	}
	return out, nil
}

// CountAllResources returns the total number of resources across all namespaces.
//
// It can return the following errors:
//
// - [api.DatabaseError].
func (s *Service) CountAllResources(ctx context.Context) (*api.ResourceCountResponse, error) {
	n, err := s.store.CountAllResources(ctx)
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("store.CountAllResources: %w", err), api.DatabaseError)
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
func (s *Service) CreateNamespace(ctx context.Context, caller *api.UserInfo, req api.NamespaceCreateRequest) (*api.NamespaceResponse, error) {
	s.mu.RLock()
	maxAttempts := s.opts.Limits.MaxNamespaceIDAttempts
	s.mu.RUnlock()

	var owner *string
	if !s.Options().Anonymous {
		if err := s.requireAuthenticated(caller); err != nil {
			return nil, err
		}
		owner = &caller.Username
	}
	for range maxAttempts {
		name, err := s.runtime.NewNamespaceID()
		if err != nil {
			return nil, api.WithErrorString(fmt.Errorf("Runtime.NewNamespaceID: %w", err), api.BadIDGeneration)
		}
		if !namespaceIDRE.MatchString(name) {
			return nil, api.WithErrorString(fmt.Errorf("%w: %q is not a valid namespace id", errBadNamespaceID, name), api.BadIDGeneration)
		}
		out, err := s.store.CreateNamespace(ctx, name, req, owner, s.runtime.Now)
		if err == nil {
			return out, nil
		}
		if errors.Is(err, backend.ErrUserNotFound) {
			// This SHOULD NEVER happen as we received the username from a user object.
			// But a race condition between a concurrent delete user call or data corruption might trigger this.
			// So we consider this a database error.
			return nil, api.WithErrorString(fmt.Errorf("%w: %w", errExistingUserNotFound, err), api.DatabaseError)
		}
		if !errors.Is(err, backend.ErrDuplicateNamespaceID) {
			return nil, api.WithErrorString(fmt.Errorf("store.CreateNamespace: %w", err), api.DatabaseError)
		}
	}
	return nil, api.WithErrorString(fmt.Errorf("%w: gave up namespace id generation after %d attempts", errInsufficientEntropy, maxAttempts), api.InsufficientEntropy)
}

// ListResources lists resources in a namespace.
//
// It can return the following errors:
//
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.InvalidNamespaceID]
// - [api.NamespaceNotFound]
// - [api.DatabaseError].
func (s *Service) ListResources(ctx context.Context, caller *api.UserInfo, params api.ListResourcesParams) (*api.PaginatedResourcesResponse, error) {
	if s.Options().Anonymous {
		if err := ValidateNamespaceID(params.Namespace); err != nil {
			return nil, api.WithErrorString(err, api.InvalidNamespaceID)
		}
	} else {
		if err := s.requireNamespaceCapability(ctx, caller, params.Namespace, canListResources); err != nil {
			return nil, err
		}
	}
	out, err := s.store.ListResources(ctx, params)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.WithErrorString(fmt.Errorf("store.UpdateResource: %w", err), api.NamespaceNotFound)
	}
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("store.UpdateResource: %w", err), api.DatabaseError)
	}
	return out, nil
}

// CreateResource creates a single resource.
//
// It can return the following errors:
//
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.InvalidNamespaceID]
// - [api.NamespaceNotFound]
// - [api.DatabaseError]
// - [api.BadIDGeneration]
// - [api.InsufficientEntropy].
func (s *Service) CreateResource(ctx context.Context, caller *api.UserInfo, namespace string, req api.ResourceCreateRequest) (*api.ResourceResponse, error) {
	if s.Options().Anonymous {
		if err := ValidateNamespaceID(namespace); err != nil {
			return nil, api.WithErrorString(err, api.InvalidNamespaceID)
		}
	} else {
		if err := s.requireNamespaceCapability(ctx, caller, namespace, canCreateResource); err != nil {
			return nil, err
		}
	}

	ns, err := s.store.GetNamespace(ctx, namespace)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.WithErrorString(fmt.Errorf("store.GetNamespace: %w", err), api.NamespaceNotFound)
	}
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("store.GetNamespace: %w", err), api.DatabaseError)
	}

	out, err := s.allocatePID(ns.PIDFormat, func(pid string) (*api.ResourceResponse, error) {
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
// - [api.InvalidNamespaceID]
// - [api.NamespaceNotFound]
// - [api.DatabaseError]
// - [api.BadIDGeneration]
// - [api.InsufficientEntropy].
func (s *Service) BatchCreateResources(ctx context.Context, caller *api.UserInfo, namespace string, reqs []api.ResourceCreateRequest) ([]api.ResourceResponse, error) {
	s.mu.RLock()
	maxBatch := s.opts.Limits.MaxBatchItems
	s.mu.RUnlock()

	if maxBatch > 0 && len(reqs) > maxBatch {
		return nil, api.WithErrorString(fmt.Errorf("%w: %d > %d", errLimitExceeded, len(reqs), maxBatch), api.ItemLimitExceeded)
	}

	if s.Options().Anonymous {
		if err := ValidateNamespaceID(namespace); err != nil {
			return nil, api.WithErrorString(err, api.InvalidNamespaceID)
		}
	} else {
		if err := s.requireNamespaceCapability(ctx, caller, namespace, canCreateResource); err != nil {
			return nil, err
		}
	}

	ns, err := s.store.GetNamespace(ctx, namespace)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.WithErrorString(fmt.Errorf("store.UpdateResource: %w", err), api.NamespaceNotFound)
	}
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("store.UpdateResource: %w", err), api.DatabaseError)
	}

	out, err := s.allocatePIDs(ns.PIDFormat, len(reqs), func(pids []string) ([]api.ResourceResponse, error) {
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
// - [api.Forbidden]
// - [api.InvalidNamespaceID]
// - [api.InvalidPID]
// - [api.NamespaceNotFound]
// - [api.ResourceNotFound]
// - [api.DatabaseError].
func (s *Service) GetResource(ctx context.Context, caller *api.UserInfo, namespace, resourcePID string) (*api.ResourceResponse, error) {
	if err := ValidatePID(resourcePID); err != nil {
		return nil, api.WithErrorString(err, api.InvalidPID)
	}
	if s.Options().Anonymous {
		if err := ValidateNamespaceID(namespace); err != nil {
			return nil, api.WithErrorString(err, api.InvalidNamespaceID)
		}
	} else if err := ValidateNamespaceID(namespace); err != nil {
		return nil, api.WithErrorString(err, api.InvalidNamespaceID)
	}

	out, err := s.store.GetResource(ctx, namespace, resourcePID)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.WithErrorString(fmt.Errorf("store.UpdateResource: %w", err), api.NamespaceNotFound)
	}
	if errors.Is(err, backend.ErrResourceNotFound) {
		return nil, api.WithErrorString(fmt.Errorf("store.UpdateResource: %w", err), api.ResourceNotFound)
	}
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("store.UpdateResource: %w", err), api.DatabaseError)
	}

	if s.Options().Anonymous {
		return out, nil
	}

	if out.Deleted {
		if caller == nil {
			return nil, api.WithErrorString(errResourceGone, api.ResourceGone)
		}
		if caller.Superuser {
			return out, nil
		}
		level, err := s.effectivePermission(ctx, caller, namespace)
		if err != nil {
			return nil, err
		}
		if canReadDeletedResource(level) {
			return out, nil
		}
		return nil, api.WithErrorString(errResourceGone, api.ResourceGone)
	}

	if caller == nil {
		return nil, api.WithErrorString(errUnauthorized, api.Unauthorized)
	}
	return out, nil
}

// UpdateResource updates a resource by namespace and pid.
//
// It can return the following errors:
//
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.InvalidNamespaceID]
// - [api.InvalidPID]
// - [api.DatabaseError]
// - [api.NamespaceNotFound]
// - [api.ResourceNotFound].
func (s *Service) UpdateResource(ctx context.Context, caller *api.UserInfo, namespace, resourcePID string, req api.ResourceUpdateRequest) (*api.ResourceResponse, error) {
	if err := ValidatePID(resourcePID); err != nil {
		return nil, api.WithErrorString(err, api.InvalidPID)
	}
	if s.Options().Anonymous {
		if err := ValidateNamespaceID(namespace); err != nil {
			return nil, api.WithErrorString(err, api.InvalidNamespaceID)
		}
	} else {
		if err := s.requireNamespaceCapability(ctx, caller, namespace, canUpdateResource); err != nil {
			return nil, err
		}
	}

	out, err := s.store.UpdateResource(ctx, namespace, resourcePID, req, s.runtime.Now)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.WithErrorString(fmt.Errorf("store.UpdateResource: %w", err), api.NamespaceNotFound)
	}
	if errors.Is(err, backend.ErrResourceNotFound) {
		return nil, api.WithErrorString(fmt.Errorf("store.UpdateResource: %w", err), api.ResourceNotFound)
	}
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("store.UpdateResource: %w", err), api.DatabaseError)
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
func (s *Service) allocatePIDs(format pid.Format, n int, insert func([]string) ([]api.ResourceResponse, error)) ([]api.ResourceResponse, error) {
	if n == 0 {
		return []api.ResourceResponse{}, nil
	}

	s.mu.RLock()
	maxAttempts := s.opts.Limits.MaxPIDAttempts
	s.mu.RUnlock()

	for range maxAttempts {
		pids := make([]string, n)
		seen := make(map[string]struct{}, n)
		for i := range n {
			// Ensure uniqueness within this batch.
			for range maxAttempts {
				candidate, err := s.runtime.NewPID(format)
				if err != nil {
					return nil, api.WithErrorString(fmt.Errorf("Runtime.NewPID: %w", err), api.BadIDGeneration)
				}
				if _, exists := seen[candidate]; exists {
					continue
				}
				seen[candidate] = struct{}{}
				pids[i] = candidate
				break
			}
			if !format.IsValid(pids[i]) {
				return nil, api.WithErrorString(fmt.Errorf("%w: %q is not a valid pid", errBadPID, pids[i]), api.BadIDGeneration)
			}
		}

		out, err := insert(pids)
		if err == nil {
			return out, nil
		}
		if !errors.Is(err, backend.ErrPIDAllocationFailed) {
			return nil, api.WithErrorString(err, api.DatabaseError)
		}
	}
	return nil, api.WithErrorString(fmt.Errorf("%w: gave up pid generation after %d attempts", errInsufficientEntropy, maxAttempts), api.InsufficientEntropy)
}

// allocatePID is like allocatePIDs but for a single PID.
func (s *Service) allocatePID(format pid.Format, insert func(string) (*api.ResourceResponse, error)) (*api.ResourceResponse, error) {
	pids, err := s.allocatePIDs(format, 1, func(pids []string) ([]api.ResourceResponse, error) {
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
