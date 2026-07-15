//spellchecker:words service
package service

//spellchecker:words context errors github bicpid backend
import (
	"context"
	"errors"
	"fmt"

	"github.com/tkw1536/bicpid/api"
	"github.com/tkw1536/bicpid/backend"
)

// canReadNamespaceMetadata checks if the given user can read namespace metadata.
func canReadNamespaceMetadata(level api.PermissionLevel) bool {
	switch level {
	case api.PermissionLevelContributor, api.PermissionLevelEditor, api.PermissionLevelManager:
		return true
	case api.PermissionLevelNone:
		return false
	default:
		return false
	}
}

// canReadDeletedResource checks if the given user can read a deleted resource.
func canReadDeletedResource(level api.PermissionLevel) bool {
	switch level {
	case api.PermissionLevelEditor, api.PermissionLevelManager:
		return true
	case api.PermissionLevelContributor, api.PermissionLevelNone:
		return false
	default:
		return false
	}
}

func canCreateResource(level api.PermissionLevel) bool {
	switch level {
	case api.PermissionLevelContributor, api.PermissionLevelEditor, api.PermissionLevelManager:
		return true
	case api.PermissionLevelNone:
		return false
	default:
		return false
	}
}

func canListResources(level api.PermissionLevel) bool {
	switch level {
	case api.PermissionLevelEditor, api.PermissionLevelManager:
		return true
	case api.PermissionLevelNone, api.PermissionLevelContributor:
		return false
	default:
		return false
	}
}

func canUpdateResource(level api.PermissionLevel) bool {
	return canListResources(level)
}

func canManagePermissions(level api.PermissionLevel) bool {
	return level == api.PermissionLevelManager
}

func (s *Service) requireAuthEnabled() error {
	if s.Options().Anonymous {
		return api.WithErrorString(errUnavailableInAnonymousMode, api.UnavailableInAnonymousMode)
	}
	return nil
}

func (s *Service) requireAuthenticated(caller *api.UserInfo) error {
	if caller == nil {
		return errUnauthorized
	}
	return nil
}

func (s *Service) effectivePermission(ctx context.Context, caller *api.UserInfo, namespace string) (api.PermissionLevel, error) {
	if caller.Superuser {
		return api.PermissionLevelManager, nil
	}
	level, err := s.store.GetNamespacePermission(ctx, namespace, caller.Username)
	if err != nil {
		return level, fmt.Errorf("Store.GetNamespacePermission: %w", err)
	}
	return level, nil
}

func (s *Service) requireNamespaceCapability(ctx context.Context, caller *api.UserInfo, namespace string, allowed func(api.PermissionLevel) bool) error {
	if err := s.requireAuthenticated(caller); err != nil {
		return err
	}
	if err := ValidateNamespaceID(namespace); err != nil {
		return err
	}

	level, err := s.effectivePermission(ctx, caller, namespace)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return err
	}
	if err != nil {
		return err
	}
	if !allowed(level) {
		return errForbidden
	}
	return nil
}

func mapAuthorizationBackendError(err error) (error, bool) {
	if errors.Is(err, backend.ErrInvalidPermissionLevel) {
		return api.WithErrorString(err, api.InvalidPermissionLevel), true
	}
	if errors.Is(err, backend.ErrPermissionNotFound) {
		return api.WithErrorString(err, api.PermissionNotFound), true
	}
	return nil, false
}

// GetNamespacePermission returns a user's permission level in a namespace.
//
// Callers may read their own permission; reading another user's permission requires manager.
func (s *Service) GetNamespacePermission(ctx context.Context, caller *api.UserInfo, namespace, username string) (*api.NamespacePermission, error) {
	if err := s.requireAuthEnabled(); err != nil {
		return nil, err
	}
	if err := s.requireAuthenticated(caller); err != nil {
		return nil, api.WithErrorString(err, api.Unauthorized)
	}
	if err := ValidateNamespaceID(namespace); err != nil {
		return nil, api.WithErrorString(err, api.InvalidNamespaceID)
	}
	if err := ValidateUsername(username); err != nil {
		return nil, api.WithErrorString(err, api.InvalidUsername)
	}

	if username != caller.Username {
		if err := s.requireNamespaceCapability(ctx, caller, namespace, canManagePermissions); err != nil {
			if errors.Is(err, errForbidden) {
				return nil, api.WithErrorString(err, api.Forbidden)
			}
			if errors.Is(err, errUnauthorized) {
				return nil, api.WithErrorString(err, api.Unauthorized)
			}
			if errors.Is(err, backend.ErrNamespaceNotFound) {
				return nil, api.WithErrorString(err, api.NamespaceNotFound)
			}
			return nil, api.WithErrorString(err, api.DatabaseError)
		}
	}

	level, err := s.store.GetNamespacePermission(ctx, namespace, username)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.WithErrorString(fmt.Errorf("Store.GetNamespacePermission: %w", err), api.NamespaceNotFound)
	}
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("Store.GetNamespacePermission: %w", err), api.DatabaseError)
	}
	return &api.NamespacePermission{Username: username, Level: level}, nil
}

// ListNamespacePermissions lists explicit permissions in a namespace. Caller must be a manager.
func (s *Service) ListNamespacePermissions(ctx context.Context, caller *api.UserInfo, namespace string, params api.ListNamespacePermissionsParams) (*api.PaginatedNamespacePermissionsResponse, error) {
	if err := s.requireAuthEnabled(); err != nil {
		return nil, err
	}
	if err := s.requireNamespaceCapability(ctx, caller, namespace, canManagePermissions); err != nil {
		if errors.Is(err, errForbidden) {
			return nil, api.WithErrorString(fmt.Errorf("Store.requireNamespaceCapability: %w", err), api.Forbidden)
		}
		if errors.Is(err, errUnauthorized) {
			return nil, api.WithErrorString(fmt.Errorf("Store.requireNamespaceCapability: %w", err), api.Unauthorized)
		}
		if errors.Is(err, backend.ErrNamespaceNotFound) {
			return nil, api.WithErrorString(fmt.Errorf("Store.requireNamespaceCapability: %w", err), api.NamespaceNotFound)
		}
		return nil, api.WithErrorString(fmt.Errorf("Store.requireNamespaceCapability: %w", err), api.DatabaseError)
	}

	page, err := s.store.ListNamespacePermissions(ctx, namespace, params)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.WithErrorString(fmt.Errorf("Store.ListNamespacePermissions: %w", err), api.NamespaceNotFound)
	}
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("Store.ListNamespacePermissions: %w", err), api.DatabaseError)
	}
	return page, nil
}

// SetNamespacePermission sets a user's permission level in a namespace. Caller must be a manager.
func (s *Service) SetNamespacePermission(ctx context.Context, caller *api.UserInfo, namespace, username string, req api.SetNamespacePermissionRequest) (*api.NamespacePermission, error) {
	if err := s.requireAuthEnabled(); err != nil {
		return nil, err
	}
	if err := ValidateUsername(username); err != nil {
		return nil, api.WithErrorString(err, api.InvalidUsername)
	}
	if err := s.requireNamespaceCapability(ctx, caller, namespace, canManagePermissions); err != nil {
		if errors.Is(err, errForbidden) {
			return nil, api.WithErrorString(err, api.Forbidden)
		}
		if errors.Is(err, errUnauthorized) {
			return nil, api.WithErrorString(err, api.Unauthorized)
		}
		if errors.Is(err, backend.ErrNamespaceNotFound) {
			return nil, api.WithErrorString(err, api.NamespaceNotFound)
		}
		return nil, api.WithErrorString(err, api.DatabaseError)
	}
	if username == caller.Username && !caller.Superuser {
		return nil, api.WithErrorString(errForbidden, api.Forbidden)
	}
	if req.Level == api.PermissionLevelNone {
		return nil, api.WithErrorString(backend.ErrInvalidPermissionLevel, api.InvalidPermissionLevel)
	}

	if err := s.store.SetNamespacePermission(ctx, namespace, username, req.Level); err != nil {
		if mapped, ok := mapAuthorizationBackendError(err); ok {
			return nil, fmt.Errorf("Store.SetNamespacePermission: %w", mapped)
		}
		if errors.Is(err, backend.ErrNamespaceNotFound) {
			return nil, api.WithErrorString(fmt.Errorf("Store.SetNamespacePermission: %w", err), api.NamespaceNotFound)
		}
		return nil, api.WithErrorString(fmt.Errorf("Store.SetNamespacePermission: %w", err), api.DatabaseError)
	}

	level, err := s.store.GetNamespacePermission(ctx, namespace, username)
	if err != nil {
		if errors.Is(err, backend.ErrNamespaceNotFound) {
			return nil, api.WithErrorString(fmt.Errorf("Store.GetNamespacePermission: %w", err), api.NamespaceNotFound)
		}
		return nil, api.WithErrorString(fmt.Errorf("Store.GetNamespacePermission: %w", err), api.DatabaseError)
	}
	return &api.NamespacePermission{Username: username, Level: level}, nil
}

// DeleteNamespacePermission removes an explicit permission record. Caller must be a manager.
func (s *Service) DeleteNamespacePermission(ctx context.Context, caller *api.UserInfo, namespace, username string) error {
	if err := s.requireAuthEnabled(); err != nil {
		return err
	}
	if err := ValidateUsername(username); err != nil {
		return api.WithErrorString(err, api.InvalidUsername)
	}
	if _, err := s.store.GetNamespace(ctx, namespace); errors.Is(err, backend.ErrNamespaceNotFound) {
		return api.WithErrorString(fmt.Errorf("Store.GetNamespace: %w", err), api.NamespaceNotFound)
	} else if err != nil {
		return api.WithErrorString(fmt.Errorf("Store.GetNamespace: %w", err), api.DatabaseError)
	}
	if err := s.requireNamespaceCapability(ctx, caller, namespace, canManagePermissions); err != nil {
		if errors.Is(err, errForbidden) {
			return api.WithErrorString(err, api.Forbidden)
		}
		if errors.Is(err, errUnauthorized) {
			return api.WithErrorString(err, api.Unauthorized)
		}
		if errors.Is(err, backend.ErrNamespaceNotFound) {
			return api.WithErrorString(err, api.NamespaceNotFound)
		}
		return api.WithErrorString(err, api.DatabaseError)
	}
	if username == caller.Username && !caller.Superuser {
		return api.WithErrorString(errForbidden, api.Forbidden)
	}

	if err := s.store.DeleteNamespacePermission(ctx, namespace, username); err != nil {
		if mapped, ok := mapAuthorizationBackendError(err); ok {
			return fmt.Errorf("Store.DeleteNamespacePermission: %w", mapped)
		}
		return api.WithErrorString(fmt.Errorf("Store.DeleteNamespacePermission: %w", err), api.DatabaseError)
	}
	return nil
}
