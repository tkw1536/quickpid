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

// canReadNamespaceMetadata checks if the given user can read namespace metadata.
func canReadNamespaceMetadata(role api.Role) bool {
	switch role {
	case api.RoleContributor, api.RoleEditor, api.RoleManager:
		return true
	case api.RoleNone:
		return false
	default:
		return false
	}
}

// canReadDeletedResource checks if the given user can read a deleted resource.
func canReadDeletedResource(role api.Role) bool {
	switch role {
	case api.RoleEditor, api.RoleManager:
		return true
	case api.RoleContributor, api.RoleNone:
		return false
	default:
		return false
	}
}

func canCreateResource(role api.Role) bool {
	switch role {
	case api.RoleContributor, api.RoleEditor, api.RoleManager:
		return true
	case api.RoleNone:
		return false
	default:
		return false
	}
}

func canListResources(role api.Role) bool {
	switch role {
	case api.RoleEditor, api.RoleManager:
		return true
	case api.RoleNone, api.RoleContributor:
		return false
	default:
		return false
	}
}

func canUpdateResource(role api.Role) bool {
	return canListResources(role)
}

func canManageRoles(role api.Role) bool {
	return role == api.RoleManager
}

// requireAuthenticated reports whether caller is authenticated.
// A nil error return guarantees that caller is not nil.
//
// It can return the following errors:
//
// - [api.Unauthorized].
func requireAuthenticated(caller *api.ValidUserInfo) error {
	if caller == nil {
		return api.WithErrorString(errUnauthorized, api.Unauthorized)
	}
	return nil
}

// effectiveRole returns the caller's role in namespace.
//
// It can return the following errors:
//
// - [api.NamespaceNotFound]
// - [api.DatabaseError].
func (s *Service) effectiveRole(ctx context.Context, caller api.ValidUserInfo, namespace api.ValidNamespaceID) (api.Role, error) {
	if caller.Superuser {
		return api.RoleManager, nil
	}

	role, err := s.store.GetNamespaceRole(ctx, namespace, caller.Username)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return role, api.WithErrorString(fmt.Errorf("Store.GetNamespaceRole: %w", err), api.NamespaceNotFound)
	}
	if err != nil {
		return role, api.WithErrorString(fmt.Errorf("Store.GetNamespaceRole: %w", err), api.DatabaseError)
	}
	return role, nil
}

// requireNamespaceCapability checks that caller is authenticated and passes the given capability check.
//
// It can return the following errors:
//
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.NamespaceNotFound]
// - [api.DatabaseError].
func (s *Service) requireNamespaceCapability(ctx context.Context, caller api.ValidUserInfo, namespace api.ValidNamespaceID, capability func(api.Role) bool) error {
	role, err := s.effectiveRole(ctx, caller, namespace)
	if err != nil {
		return err
	}
	if !capability(role) {
		return api.WithErrorString(errForbidden, api.Forbidden)
	}
	return nil
}

// mapAuthorizationBackendError translates backend store errors to API-annotated errors.
func mapAuthorizationBackendError(err error) (error, bool) {
	if errors.Is(err, backend.ErrInvalidRole) {
		return api.WithErrorString(err, api.InvalidRole), true
	}
	if errors.Is(err, backend.ErrRoleNotFound) {
		return api.WithErrorString(err, api.RoleNotFound), true
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
func (s *Service) GetNamespaceRole(ctx context.Context, caller api.ValidUserInfo, namespace api.ValidNamespaceID, username api.ValidUsername) (*api.NamespaceRole, error) {
	if username.String() != caller.Username.String() {
		if err := s.requireNamespaceCapability(ctx, caller, namespace, canManageRoles); err != nil {
			return nil, err
		}
	}

	role, err := s.store.GetNamespaceRole(ctx, namespace, username)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.WithErrorString(fmt.Errorf("Store.GetNamespaceRole: %w", err), api.NamespaceNotFound)
	}
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("Store.GetNamespaceRole: %w", err), api.DatabaseError)
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
func (s *Service) ListNamespaceRoles(ctx context.Context, caller api.ValidUserInfo, namespace api.ValidNamespaceID, params api.ListNamespaceRolesParams) (*api.PaginatedNamespaceRolesResponse, error) {
	if err := s.requireNamespaceCapability(ctx, caller, namespace, canManageRoles); err != nil {
		return nil, err
	}

	page, err := s.store.ListNamespaceRoles(ctx, namespace, params)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.WithErrorString(fmt.Errorf("Store.ListNamespaceRoles: %w", err), api.NamespaceNotFound)
	}
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("Store.ListNamespaceRoles: %w", err), api.DatabaseError)
	}
	return page, nil
}

// ListUserRoles lists explicit roles for a user across namespaces.
//
// Callers may list their own roles; listing another user's roles requires a superuser.
//
// It can return the following errors:
//
// - [api.UnavailableInAnonymousMode]
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.UserNotFound]
// - [api.DatabaseError].
func (s *Service) ListUserRoles(ctx context.Context, caller api.ValidUserInfo, target *api.ValidUsername, params api.ListUserRolesParams) (*api.PaginatedUserRolesResponse, error) {
	username, err := resolveTarget(caller, target)
	if err != nil {
		return nil, err
	}

	page, err := s.store.ListUserRoles(ctx, username, params)
	if mapped, ok := mapAuthBackendError(err); ok {
		return nil, fmt.Errorf("Store.ListUserRoles: %w", mapped)
	}
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("Store.ListUserRoles: %w", err), api.DatabaseError)
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
func (s *Service) SetNamespaceRole(ctx context.Context, caller api.ValidUserInfo, namespace api.ValidNamespaceID, username api.ValidUsername, req api.SetNamespaceRoleRequest) (*api.NamespaceRole, error) {
	if err := s.requireNamespaceCapability(ctx, caller, namespace, canManageRoles); err != nil {
		return nil, err
	}
	if username.String() == caller.Username.String() && !caller.Superuser {
		return nil, api.WithErrorString(errForbidden, api.Forbidden)
	}
	if req.Role == api.RoleNone {
		return nil, api.WithErrorString(backend.ErrInvalidRole, api.InvalidRole)
	}

	if err := s.store.SetNamespaceRole(ctx, namespace, username, req.Role); err != nil {
		if mapped, ok := mapAuthorizationBackendError(err); ok {
			return nil, fmt.Errorf("Store.SetNamespaceRole: %w", mapped)
		}
		if errors.Is(err, backend.ErrNamespaceNotFound) {
			return nil, api.WithErrorString(fmt.Errorf("Store.SetNamespaceRole: %w", err), api.NamespaceNotFound)
		}
		return nil, api.WithErrorString(fmt.Errorf("Store.SetNamespaceRole: %w", err), api.DatabaseError)
	}

	role, err := s.store.GetNamespaceRole(ctx, namespace, username)
	if err != nil {
		if errors.Is(err, backend.ErrNamespaceNotFound) {
			return nil, api.WithErrorString(fmt.Errorf("Store.GetNamespaceRole: %w", err), api.NamespaceNotFound)
		}
		return nil, api.WithErrorString(fmt.Errorf("Store.GetNamespaceRole: %w", err), api.DatabaseError)
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
func (s *Service) DeleteNamespaceRole(ctx context.Context, caller api.ValidUserInfo, namespace api.ValidNamespaceID, username api.ValidUsername) error {
	if _, err := s.store.GetNamespace(ctx, namespace); errors.Is(err, backend.ErrNamespaceNotFound) {
		return api.WithErrorString(fmt.Errorf("Store.GetNamespace: %w", err), api.NamespaceNotFound)
	} else if err != nil {
		return api.WithErrorString(fmt.Errorf("Store.GetNamespace: %w", err), api.DatabaseError)
	}
	if err := s.requireNamespaceCapability(ctx, caller, namespace, canManageRoles); err != nil {
		return err
	}
	if username.String() == caller.Username.String() && !caller.Superuser {
		return api.WithErrorString(errForbidden, api.Forbidden)
	}

	if err := s.store.DeleteNamespaceRole(ctx, namespace, username); err != nil {
		if mapped, ok := mapAuthorizationBackendError(err); ok {
			return fmt.Errorf("Store.DeleteNamespaceRole: %w", mapped)
		}
		return api.WithErrorString(fmt.Errorf("Store.DeleteNamespaceRole: %w", err), api.DatabaseError)
	}
	return nil
}
