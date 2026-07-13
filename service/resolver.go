//spellchecker:words service
package service

//spellchecker:words context errors regexp github bicpid backend
import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/tkw1536/bicpid/api"
	"github.com/tkw1536/bicpid/backend"
	"github.com/tkw1536/bicpid/pid"
)

var (
	namespaceIDRE = regexp.MustCompile(`^[a-z0-9_-]+$`)
	pidRE         = regexp.MustCompile(`^[a-z0-9_-]+$`)
	usernameRE    = regexp.MustCompile(`^[a-z0-9_-]+$`)
)

// ValidateNamespaceID reports whether id is a valid namespace identifier.
func ValidateNamespaceID(id string) error {
	if !namespaceIDRE.MatchString(id) {
		return errInvalidNamespaceID
	}
	return nil
}

// ValidatePID reports whether id is a valid pid.
func ValidatePID(id string) error {
	if !pidRE.MatchString(id) {
		return errInvalidPID
	}
	return nil
}

// ValidateUsername reports whether username is a valid username.
func ValidateUsername(username string) error {
	if !usernameRE.MatchString(username) {
		return errInvalidUsername
	}
	return nil
}

func mapCapabilityError(err error) (api.Error, error) {
	if errors.Is(err, errForbidden) || errors.Is(err, errUnauthorized) {
		return "", err
	}
	if errors.Is(err, errInvalidNamespaceID) {
		return api.InvalidNamespaceID, err
	}
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return api.NamespaceNotFound, err
	}
	return api.DatabaseError, err
}

var errExistingUserNotFound = errors.New("existing user not found")

// ListNamespaces lists namespaces.
//
// It can return the following errors:
//
// - [api.Unauthorized]
// - [api.DatabaseError]
func (s *Service) ListNamespaces(ctx context.Context, caller *api.UserInfo, params api.ListNamespacesParams) (*api.PaginatedNamespacesResponse, api.Error, error) {
	if !s.Options().Anonymous {
		if err := s.requireAuthenticated(caller); err != nil {
			return nil, "", err
		}
	}

	var user *string
	if caller != nil && !caller.Superuser {
		user = new(caller.Username)
	}
	out, err := s.resolver.ListNamespaces(ctx, user, params)
	if errors.Is(err, backend.ErrUserNotFound) {
		// This SHOULD NEVER happen as we received the username from a user object.
		// But a race condition between a concurrent delete user call or data corruption might trigger this.
		// So we consider this a database error.
		return nil, api.DatabaseError, fmt.Errorf("%w: %w", errExistingUserNotFound, err)
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}
	return out, "", nil
}

// GetNamespace returns one namespace by id.
//
// It can return the following errors:
//
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.InvalidNamespaceID]
// - [api.NamespaceNotFound]
// - [api.DatabaseError]
func (s *Service) GetNamespace(ctx context.Context, caller *api.UserInfo, namespace string) (*api.NamespaceResponse, api.Error, error) {
	if s.Options().Anonymous {
		if err := ValidateNamespaceID(namespace); err != nil {
			return nil, api.InvalidNamespaceID, err
		}
	} else {
		if err := s.requireNamespaceCapability(ctx, caller, namespace, canReadNamespace); err != nil {
			specError, mapped := mapCapabilityError(err)
			return nil, specError, mapped
		}
	}
	out, err := s.resolver.GetNamespace(ctx, namespace)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.NamespaceNotFound, err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}
	return out, "", nil
}

// CountAllResources returns the total number of resources across all namespaces.
//
// It can return the following errors:
//
// - [api.DatabaseError]
func (s *Service) CountAllResources(ctx context.Context) (*api.ResourceCountResponse, api.Error, error) {
	n, err := s.resolver.CountAllResources(ctx)
	if err != nil {
		return nil, api.DatabaseError, err
	}
	return &api.ResourceCountResponse{Total: int(n)}, "", nil
}

// CreateNamespace creates a new namespace owned by the authenticated caller.
//
// It can return the following errors:
//
// - [api.Unauthorized]
// - [api.DatabaseError]
// - [api.BadIDGeneration]
// - [api.InsufficientEntropy]
func (s *Service) CreateNamespace(ctx context.Context, caller *api.UserInfo, req api.NamespaceCreateRequest) (*api.NamespaceResponse, api.Error, error) {
	s.mu.RLock()
	maxAttempts := s.opts.Limits.MaxNamespaceIDAttempts
	s.mu.RUnlock()

	var owner *string
	if !s.Options().Anonymous {
		if err := s.requireAuthenticated(caller); err != nil {
			return nil, "", err
		}
		owner = &caller.Username
	}
	for range maxAttempts {
		name, err := s.runtime.NewNamespaceID()
		if err != nil {
			return nil, api.BadIDGeneration, err
		}
		if !namespaceIDRE.MatchString(name) {
			return nil, api.BadIDGeneration, fmt.Errorf("%w: %q is not a valid namespace id", errBadNamespaceID, name)
		}
		out, err := s.resolver.CreateNamespace(ctx, name, req, owner, s.runtime.Now)
		if err == nil {
			return out, "", nil
		}
		if errors.Is(err, backend.ErrUserNotFound) {
			// This SHOULD NEVER happen as we received the username from a user object.
			// But a race condition between a concurrent delete user call or data corruption might trigger this.
			// So we consider this a database error.
			return nil, api.DatabaseError, fmt.Errorf("%w: %w", errExistingUserNotFound, err)
		}
		if !errors.Is(err, backend.ErrDuplicateNamespaceID) {
			return nil, api.DatabaseError, err
		}
	}
	return nil, api.InsufficientEntropy, fmt.Errorf("%w: gave up namespace id generation after %d attempts", errInsufficientEntropy, maxAttempts)
}

// ListResources lists resources in a namespace.
//
// It can return the following errors:
//
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.InvalidNamespaceID]
// - [api.NamespaceNotFound]
// - [api.DatabaseError]
func (s *Service) ListResources(ctx context.Context, caller *api.UserInfo, params api.ListResourcesParams) (*api.PaginatedResourcesResponse, api.Error, error) {
	if s.Options().Anonymous {
		if err := ValidateNamespaceID(params.Namespace); err != nil {
			return nil, api.InvalidNamespaceID, err
		}
	} else {
		if err := s.requireNamespaceCapability(ctx, caller, params.Namespace, canListResources); err != nil {
			specError, mapped := mapCapabilityError(err)
			return nil, specError, mapped
		}
	}
	out, err := s.resolver.ListResources(ctx, params)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.NamespaceNotFound, err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}
	return out, "", nil
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
// - [api.InsufficientEntropy]
func (s *Service) CreateResource(ctx context.Context, caller *api.UserInfo, namespace string, req api.ResourceCreateRequest) (*api.ResourceResponse, api.Error, error) {
	if s.Options().Anonymous {
		if err := ValidateNamespaceID(namespace); err != nil {
			return nil, api.InvalidNamespaceID, err
		}
	} else {
		if err := s.requireNamespaceCapability(ctx, caller, namespace, canCreateResource); err != nil {
			specError, mapped := mapCapabilityError(err)
			return nil, specError, mapped
		}
	}

	ns, err := s.resolver.GetNamespace(ctx, namespace)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.NamespaceNotFound, err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}

	out, specError, err := s.allocatePID(ns.PIDFormat, func(pid string) (*api.ResourceResponse, error) {
		return s.resolver.CreateResource(ctx, namespace, pid, req, s.runtime.Now)
	})
	if err != nil {
		return nil, specError, err
	}
	return out, "", nil
}

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
// - [api.InsufficientEntropy]
func (s *Service) BatchCreateResources(ctx context.Context, caller *api.UserInfo, namespace string, reqs []api.ResourceCreateRequest) ([]api.ResourceResponse, api.Error, error) {
	s.mu.RLock()
	maxBatch := s.opts.Limits.MaxBatchItems
	s.mu.RUnlock()

	if maxBatch > 0 && len(reqs) > maxBatch {
		return nil, api.ItemLimitExceeded, fmt.Errorf("%d > %d", len(reqs), maxBatch)
	}

	if s.Options().Anonymous {
		if err := ValidateNamespaceID(namespace); err != nil {
			return nil, api.InvalidNamespaceID, err
		}
	} else {
		if err := s.requireNamespaceCapability(ctx, caller, namespace, canCreateResource); err != nil {
			specError, mapped := mapCapabilityError(err)
			return nil, specError, mapped
		}
	}

	ns, err := s.resolver.GetNamespace(ctx, namespace)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.NamespaceNotFound, err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}

	out, specError, err := s.allocatePIDs(ns.PIDFormat, len(reqs), func(pids []string) ([]api.ResourceResponse, error) {
		return s.resolver.BatchCreateResources(ctx, namespace, pids, reqs, s.runtime.Now)
	})
	if err != nil {
		return nil, specError, err
	}
	return out, "", nil
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
// - [api.DatabaseError]
func (s *Service) GetResource(ctx context.Context, caller *api.UserInfo, namespace, resourcePID string) (*api.ResourceResponse, api.Error, error) {
	if err := ValidatePID(resourcePID); err != nil {
		return nil, api.InvalidPID, err
	}
	if s.Options().Anonymous {
		if err := ValidateNamespaceID(namespace); err != nil {
			return nil, api.InvalidNamespaceID, err
		}
	} else {
		if err := s.requireNamespaceCapability(ctx, caller, namespace, canReadNamespace); err != nil {
			specError, mapped := mapCapabilityError(err)
			return nil, specError, mapped
		}
	}

	out, err := s.resolver.GetResource(ctx, namespace, resourcePID)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.NamespaceNotFound, err
	}
	if errors.Is(err, backend.ErrResourceNotFound) {
		return nil, api.ResourceNotFound, err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}
	return out, "", nil
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
// - [api.ResourceNotFound]
func (s *Service) UpdateResource(ctx context.Context, caller *api.UserInfo, namespace, resourcePID string, req api.ResourceUpdateRequest) (*api.ResourceResponse, api.Error, error) {
	if err := ValidatePID(resourcePID); err != nil {
		return nil, api.InvalidPID, err
	}
	if s.Options().Anonymous {
		if err := ValidateNamespaceID(namespace); err != nil {
			return nil, api.InvalidNamespaceID, err
		}
	} else {
		if err := s.requireNamespaceCapability(ctx, caller, namespace, canUpdateResource); err != nil {
			specError, mapped := mapCapabilityError(err)
			return nil, specError, mapped
		}
	}

	out, err := s.resolver.UpdateResource(ctx, namespace, resourcePID, req, s.runtime.Now)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.NamespaceNotFound, err
	}
	if errors.Is(err, backend.ErrResourceNotFound) {
		return nil, api.ResourceNotFound, err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}
	return out, "", nil
}

// allocatePIDs allocates n unique pids in the given namespace.
//
// It can return the following errors:
//
// - [api.BadIDGeneration]
// - [api.DatabaseError]
// - [api.InsufficientEntropy]
func (s *Service) allocatePIDs(format pid.Format, n int, insert func([]string) ([]api.ResourceResponse, error)) ([]api.ResourceResponse, api.Error, error) {
	if n == 0 {
		return []api.ResourceResponse{}, "", nil
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
					return nil, api.BadIDGeneration, err
				}
				if _, exists := seen[candidate]; exists {
					continue
				}
				seen[candidate] = struct{}{}
				pids[i] = candidate
				break
			}
			if !format.IsValid(pids[i]) {
				return nil, api.BadIDGeneration, fmt.Errorf("%w: %q is not a valid pid", errBadPID, pids[i])
			}
		}

		out, err := insert(pids)
		if err == nil {
			return out, "", nil
		}
		if !errors.Is(err, backend.ErrPIDAllocationFailed) {
			return nil, api.DatabaseError, err
		}
	}
	return nil, api.InsufficientEntropy, fmt.Errorf("%w: gave up pid generation after %d attempts", errInsufficientEntropy, maxAttempts)
}

// allocatePID is like allocatePIDs but for a single PID.
func (s *Service) allocatePID(format pid.Format, insert func(string) (*api.ResourceResponse, error)) (*api.ResourceResponse, api.Error, error) {
	pids, specError, err := s.allocatePIDs(format, 1, func(pids []string) ([]api.ResourceResponse, error) {
		res, err := insert(pids[0])
		if err != nil {
			return nil, err
		}
		return []api.ResourceResponse{*res}, nil
	})
	if err != nil {
		return nil, specError, err
	}
	p := pids[0]
	return &p, "", nil
}
