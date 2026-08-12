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

// requireAuthenticated reports whether caller is authenticated.
// A nil error return guarantees that caller is not nil.
//
// It can return the following errors:
//
// - [api.Unauthorized].
func requireAuthenticated(caller *api.Caller) error {
	if caller == nil {
		return api.WithErrorCode(errUnauthorized, api.Unauthorized)
	}
	return nil
}

// mapAuthorizationBackendError translates backend store errors to API-annotated errors.
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
// Callers may read their own role; reading another user's role requires manager.
//
// It can return the following errors:
//
// - [api.UnavailableInAnonymousMode]
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.NamespaceNotFound]
// - [api.DatabaseError].
func (s *Service) GetNamespaceRole(ctx context.Context, caller *api.Caller, namespace api.ValidNamespaceID, username api.ValidUsername) (*api.NamespaceRole, error) {
	can, err := s.canNamespace(ctx, caller, namespace)
	if err != nil {
		return nil, err
	}

	if caller != nil && username == caller.Username() {
		if err := can.GetOwnNamespaceRole(); err != nil {
			return nil, fmt.Errorf("GetOwnNamespaceRole() check failed: %w", err)
		}
	} else {
		if err := can.GetOtherUserNamespaceRole(); err != nil {
			return nil, fmt.Errorf("GetOtherUserNamespaceRole() check failed: %w", err)
		}
	}

	role, err := s.store.GetNamespaceRole(ctx, namespace, username)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.WithErrorCode(fmt.Errorf("namespace not found: %w", err), api.NamespaceNotFound)
	}
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to get namespace role: %w", err), api.DatabaseError)
	}
	return &api.NamespaceRole{Username: username.String(), Role: role}, nil
}

// ListNamespaceRoles lists explicit roles in a namespace. Caller must be a manager.
//
// It can return the following errors:
//
// - [api.UnavailableInAnonymousMode]
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.NamespaceNotFound]
// - [api.DatabaseError].
func (s *Service) ListNamespaceRoles(ctx context.Context, caller api.Caller, namespace api.ValidNamespaceID, params api.ListNamespaceRolesParams) (*api.PaginatedNamespaceRolesResponse, error) {
	can, err := s.canNamespace(ctx, &caller, namespace)
	if err != nil {
		return nil, err
	}
	if err := can.ListNamespaceRoles(); err != nil {
		return nil, fmt.Errorf("ListNamespaceRoles() check failed: %w", err)
	}

	page, err := s.store.ListNamespaceRoles(ctx, namespace, params)
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
	can := s.canUser(&caller)
	if err := can.ListUserRoles(); err != nil {
		return nil, fmt.Errorf("ListUserRoles() check failed: %w", err)
	}

	page, err := s.store.ListUserRoles(ctx, caller.Username(), params)
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to list user roles: %w", err), api.DatabaseError)
	}
	return page, nil
}

// SetNamespaceRole sets a user's role in a namespace. Caller must be a manager.
//
// It can return the following errors:
//
// - [api.UnavailableInAnonymousMode]
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.InvalidRole]
// - [api.NamespaceNotFound]
// - [api.DatabaseError].
func (s *Service) SetNamespaceRole(ctx context.Context, caller api.Caller, namespace api.ValidNamespaceID, username api.ValidUsername, req api.SetNamespaceRoleRequest) (*api.NamespaceRole, error) {
	can, err := s.canNamespace(ctx, &caller, namespace)
	if err != nil {
		return nil, err
	}
	if username == caller.Username() {
		if err := can.SetOwnNamespaceRole(); err != nil {
			return nil, fmt.Errorf("SetOwnNamespaceRole() check failed: %w", err)
		}
	} else {
		if err := can.SetOtherNamespaceRole(); err != nil {
			return nil, fmt.Errorf("SetOtherNamespaceRole() check failed: %w", err)
		}
	}

	if req.Role == api.RoleNone {
		return nil, api.WithErrorCode(backend.ErrInvalidRole, api.InvalidRole)
	}

	if err := s.store.SetNamespaceRole(ctx, namespace, username, req.Role); err != nil {
		if mapped, ok := mapAuthorizationBackendError(err); ok {
			return nil, mapped
		}
		if errors.Is(err, backend.ErrNamespaceNotFound) {
			return nil, api.WithErrorCode(fmt.Errorf("namespace not found: %w", err), api.NamespaceNotFound)
		}
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to set namespace role: %w", err), api.DatabaseError)
	}

	role, err := s.store.GetNamespaceRole(ctx, namespace, username)
	if err != nil {
		if errors.Is(err, backend.ErrNamespaceNotFound) {
			return nil, api.WithErrorCode(fmt.Errorf("namespace not found: %w", err), api.NamespaceNotFound)
		}
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to get namespace role: %w", err), api.DatabaseError)
	}
	return &api.NamespaceRole{Username: username.String(), Role: role}, nil
}

// DeleteNamespaceRole removes an explicit role record. Caller must be a manager.
//
// It can return the following errors:
//
// - [api.UnavailableInAnonymousMode]
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.NamespaceNotFound]
// - [api.RoleNotFound]
// - [api.DatabaseError].
func (s *Service) DeleteNamespaceRole(ctx context.Context, caller api.Caller, namespace api.ValidNamespaceID, username api.ValidUsername) error {
	can, err := s.canNamespace(ctx, &caller, namespace)
	if err != nil {
		return err
	}

	if _, err := s.store.GetNamespace(ctx, namespace); errors.Is(err, backend.ErrNamespaceNotFound) {
		return api.WithErrorCode(fmt.Errorf("namespace not found: %w", err), api.NamespaceNotFound)
	} else if err != nil {
		return api.WithErrorCode(fmt.Errorf("backend failed to get namespace: %w", err), api.DatabaseError)
	}

	if username == caller.Username() {
		if err := can.ClearOwnNamespaceRole(); err != nil {
			return fmt.Errorf("ClearOwnNamespaceRole() check failed: %w", err)
		}
	} else {
		if err := can.ClearOtherNamespaceRole(); err != nil {
			return fmt.Errorf("ClearOtherNamespaceRole() check failed: %w", err)
		}
	}

	if err := s.store.DeleteNamespaceRole(ctx, namespace, username); err != nil {
		if mapped, ok := mapAuthorizationBackendError(err); ok {
			return mapped
		}
		return api.WithErrorCode(fmt.Errorf("backend failed to delete namespace role: %w", err), api.DatabaseError)
	}
	return nil
}
