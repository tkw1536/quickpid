//spellchecker:words service
package service

//spellchecker:words context errors github quickpid backend
import (
	"context"
	"errors"
	"fmt"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
)

// mapAuthorizationBackendError translates backend errors to API-annotated errors.
// TODO: Inline this where appropriate.
func mapAuthorizationBackendError(err error) (error, bool) {
	if errors.Is(err, backend.ErrInvalidRole) {
		return api.WithErrorCode(fmt.Errorf("invalid role: %w", err), api.InvalidRole), true
	}
	if errors.Is(err, backend.ErrRoleNotFound) {
		return api.WithErrorCode(fmt.Errorf("role not found: %w", err), api.RoleNotFound), true
	}
	return nil, false
}

// GetNamespaceRole returns a user's role in a namespace.
//
// It can return the following errors:
//
// - [api.NamespaceNotFound]
// - [api.DatabaseError].
func (s *Service) GetNamespaceRole(ctx context.Context, caller *api.Caller, namespace api.ValidNamespaceID, username api.ValidUsername) (*api.NamespaceRole, error) {
	role, err := s.backend.GetNamespaceRole(ctx, namespace, username)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.WithErrorCode(fmt.Errorf("namespace not found: %w", err), api.NamespaceNotFound)
	}
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to get namespace role: %w", err), api.DatabaseError)
	}
	return &api.NamespaceRole{Username: username.String(), Role: role}, nil
}

// ListNamespaceRoles lists explicit roles in a namespace.
//
// It can return the following errors:
//
// - [api.NamespaceNotFound]
// - [api.DatabaseError].
func (s *Service) ListNamespaceRoles(ctx context.Context, caller api.Caller, namespace api.ValidNamespaceID, params api.ListNamespaceRolesParams) (*api.PaginatedNamespaceRolesResponse, error) {
	page, err := s.backend.ListNamespaceRoles(ctx, namespace, params)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.WithErrorCode(fmt.Errorf("namespace not found: %w", err), api.NamespaceNotFound)
	}
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to list namespace roles: %w", err), api.DatabaseError)
	}
	return page, nil
}

// ListUserRoles lists a user's roles across all namespaces.
//
// It can return the following errors:
//
// - [api.DatabaseError].
func (s *Service) ListUserRoles(ctx context.Context, caller api.Caller, params api.ListUserRolesParams) (*api.PaginatedUserRolesResponse, error) {
	page, err := s.backend.ListUserRoles(ctx, caller.Username(), params)
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to list user roles: %w", err), api.DatabaseError)
	}
	return page, nil
}

// SetNamespaceRole sets a user's role in a namespace.
//
// It can return the following errors:
//
// - [api.InvalidRole]
// - [api.NamespaceNotFound]
// - [api.DatabaseError].
func (s *Service) SetNamespaceRole(ctx context.Context, caller api.Caller, namespace api.ValidNamespaceID, username api.ValidUsername, req api.SetNamespaceRoleRequest) (*api.NamespaceRole, error) {
	if req.Role == api.RoleNone {
		return nil, api.WithErrorCode(backend.ErrInvalidRole, api.InvalidRole)
	}

	if err := s.backend.SetNamespaceRole(ctx, namespace, username, req.Role); err != nil {
		if mapped, ok := mapAuthorizationBackendError(err); ok {
			return nil, mapped
		}
		if errors.Is(err, backend.ErrNamespaceNotFound) {
			return nil, api.WithErrorCode(fmt.Errorf("namespace not found: %w", err), api.NamespaceNotFound)
		}
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to set namespace role: %w", err), api.DatabaseError)
	}

	role, err := s.backend.GetNamespaceRole(ctx, namespace, username)
	if err != nil {
		if errors.Is(err, backend.ErrNamespaceNotFound) {
			return nil, api.WithErrorCode(fmt.Errorf("namespace not found: %w", err), api.NamespaceNotFound)
		}
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to get namespace role: %w", err), api.DatabaseError)
	}
	return &api.NamespaceRole{Username: username.String(), Role: role}, nil
}

// DeleteNamespaceRole removes an explicit role record.
//
// It can return the following errors:
//
// - [api.NamespaceNotFound]
// - [api.RoleNotFound]
// - [api.DatabaseError].
func (s *Service) DeleteNamespaceRole(ctx context.Context, caller api.Caller, namespace api.ValidNamespaceID, username api.ValidUsername) error {
	if _, err := s.backend.GetNamespace(ctx, namespace); errors.Is(err, backend.ErrNamespaceNotFound) {
		return api.WithErrorCode(fmt.Errorf("namespace not found: %w", err), api.NamespaceNotFound)
	} else if err != nil {
		return api.WithErrorCode(fmt.Errorf("backend failed to get namespace: %w", err), api.DatabaseError)
	}

	if err := s.backend.DeleteNamespaceRole(ctx, namespace, username); err != nil {
		if mapped, ok := mapAuthorizationBackendError(err); ok {
			return mapped
		}
		return api.WithErrorCode(fmt.Errorf("backend failed to delete namespace role: %w", err), api.DatabaseError)
	}
	return nil
}

// ExplicitNamespaceRole returns the user's explicit role in a namespace for scope checks.
// Returns [api.RoleNone] when no role is stored.
//
// It can return the following errors:
//
// - [api.NamespaceNotFound]
// - [api.DatabaseError].
func (s *Service) ExplicitNamespaceRole(ctx context.Context, namespace api.ValidNamespaceID, username api.ValidUsername) (api.Role, error) {
	role, err := s.backend.GetNamespaceRole(ctx, namespace, username)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return api.RoleNone, api.WithErrorCode(fmt.Errorf("namespace not found: %w", err), api.NamespaceNotFound)
	}
	if err != nil {
		return api.RoleNone, api.WithErrorCode(fmt.Errorf("backend failed to get namespace role: %w", err), api.DatabaseError)
	}
	return role, nil
}
