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
)

var (
	errExistingUserNotFound = errors.New("existing user not found")
	errBadNamespaceID       = errors.New("bad namespace id generated")
)

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
