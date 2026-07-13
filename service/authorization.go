//spellchecker:words service
package service

//spellchecker:words context errors github bicpid backend
import (
	"context"
	"errors"

	"github.com/tkw1536/bicpid/api"
	"github.com/tkw1536/bicpid/backend"
)

func permissionRank(level api.PermissionLevel) int {
	switch level {
	case api.PermissionLevelNone:
		return 0
	case api.PermissionLevelContributor:
		return 1
	case api.PermissionLevelEditor:
		return 2
	case api.PermissionLevelManager:
		return 3
	default:
		return -1
	}
}

// canReadNamespace checks if the given user can read namespace metadata.
func canReadNamespace(level api.PermissionLevel) bool {
	switch level {
	case api.PermissionLevelNone, api.PermissionLevelEditor, api.PermissionLevelManager:
		return true
	default:
		return false
	}
}

// canReadResource checks if the given user can read a non-deleted resource.
func canReadResource(level api.PermissionLevel) bool {
	switch level {
	case api.PermissionLevelNone, api.PermissionLevelEditor, api.PermissionLevelManager:
		return true
	default:
		return false
	}
}

// canReadDeletedResource checks if the given user can read a deleted resource.
func canReadDeletedResource(level api.PermissionLevel) bool {
	return canListResources(level)
}

func canCreateResource(level api.PermissionLevel) bool {
	switch level {
	case api.PermissionLevelContributor, api.PermissionLevelEditor, api.PermissionLevelManager:
		return true
	default:
		return false
	}
}

func canListResources(level api.PermissionLevel) bool {
	switch level {
	case api.PermissionLevelEditor, api.PermissionLevelManager:
		return true
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

func (s *Service) requireAuthEnabled() (api.Error, error) {
	if s.Options().Anonymous {
		return api.UnavailableInAnonymousMode, errUnavailableInAnonymousMode
	}
	return "", nil
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
	return s.store.GetNamespacePermission(ctx, namespace, caller.Username)
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

func mapAuthorizationBackendError(err error) (api.Error, bool) {
	if errors.Is(err, backend.ErrInvalidPermissionLevel) {
		return api.InvalidPermissionLevel, true
	}
	if errors.Is(err, backend.ErrPermissionNotFound) {
		return api.PermissionNotFound, true
	}
	return "", false
}

// GetNamespacePermission returns a user's permission level in a namespace.
//
// Callers may read their own permission; reading another user's permission requires manager.
func (s *Service) GetNamespacePermission(ctx context.Context, caller *api.UserInfo, namespace, username string) (*api.NamespacePermission, api.Error, error) {
	if specError, err := s.requireAuthEnabled(); err != nil {
		return nil, specError, err
	}
	if err := s.requireAuthenticated(caller); err != nil {
		return nil, "", err
	}
	if err := ValidateNamespaceID(namespace); err != nil {
		return nil, api.InvalidNamespaceID, err
	}
	if err := ValidateUsername(username); err != nil {
		return nil, api.InvalidUsername, err
	}

	if username != caller.Username {
		if err := s.requireNamespaceCapability(ctx, caller, namespace, canManagePermissions); err != nil {
			if errors.Is(err, errForbidden) || errors.Is(err, errUnauthorized) {
				return nil, "", err
			}
			if errors.Is(err, backend.ErrNamespaceNotFound) {
				return nil, api.NamespaceNotFound, err
			}
			return nil, api.DatabaseError, err
		}
	}

	level, err := s.store.GetNamespacePermission(ctx, namespace, username)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.NamespaceNotFound, err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}
	return &api.NamespacePermission{Username: username, Level: level}, "", nil
}

// ListNamespacePermissions lists explicit permissions in a namespace. Caller must be a manager.
func (s *Service) ListNamespacePermissions(ctx context.Context, caller *api.UserInfo, namespace string, params api.ListNamespacePermissionsParams) (*api.PaginatedNamespacePermissionsResponse, api.Error, error) {
	if specError, err := s.requireAuthEnabled(); err != nil {
		return nil, specError, err
	}
	if err := s.requireNamespaceCapability(ctx, caller, namespace, canManagePermissions); err != nil {
		if errors.Is(err, errForbidden) || errors.Is(err, errUnauthorized) {
			return nil, "", err
		}
		if errors.Is(err, backend.ErrNamespaceNotFound) {
			return nil, api.NamespaceNotFound, err
		}
		return nil, api.DatabaseError, err
	}

	page, err := s.store.ListNamespacePermissions(ctx, namespace, params)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.NamespaceNotFound, err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}
	return page, "", nil
}

// SetNamespacePermission sets a user's permission level in a namespace. Caller must be a manager.
func (s *Service) SetNamespacePermission(ctx context.Context, caller *api.UserInfo, namespace, username string, req api.SetNamespacePermissionRequest) (*api.NamespacePermission, api.Error, error) {
	if specError, err := s.requireAuthEnabled(); err != nil {
		return nil, specError, err
	}
	if err := ValidateUsername(username); err != nil {
		return nil, api.InvalidUsername, err
	}
	if err := s.requireNamespaceCapability(ctx, caller, namespace, canManagePermissions); err != nil {
		if errors.Is(err, errForbidden) || errors.Is(err, errUnauthorized) {
			return nil, "", err
		}
		if errors.Is(err, backend.ErrNamespaceNotFound) {
			return nil, api.NamespaceNotFound, err
		}
		return nil, api.DatabaseError, err
	}
	if username == caller.Username && !caller.Superuser {
		return nil, "", errForbidden
	}
	if req.Level == api.PermissionLevelNone {
		return nil, api.InvalidPermissionLevel, backend.ErrInvalidPermissionLevel
	}

	if err := s.store.SetNamespacePermission(ctx, namespace, username, req.Level); err != nil {
		if specError, ok := mapAuthorizationBackendError(err); ok {
			return nil, specError, err
		}
		if errors.Is(err, backend.ErrNamespaceNotFound) {
			return nil, api.NamespaceNotFound, err
		}
		return nil, api.DatabaseError, err
	}

	level, err := s.store.GetNamespacePermission(ctx, namespace, username)
	if err != nil {
		if errors.Is(err, backend.ErrNamespaceNotFound) {
			return nil, api.NamespaceNotFound, err
		}
		return nil, api.DatabaseError, err
	}
	return &api.NamespacePermission{Username: username, Level: level}, "", nil
}

// DeleteNamespacePermission removes an explicit permission record. Caller must be a manager.
func (s *Service) DeleteNamespacePermission(ctx context.Context, caller *api.UserInfo, namespace, username string) (api.Error, error) {
	if specError, err := s.requireAuthEnabled(); err != nil {
		return specError, err
	}
	if err := ValidateUsername(username); err != nil {
		return api.InvalidUsername, err
	}
	if _, err := s.store.GetNamespace(ctx, namespace); errors.Is(err, backend.ErrNamespaceNotFound) {
		return api.NamespaceNotFound, err
	} else if err != nil {
		return api.DatabaseError, err
	}
	if err := s.requireNamespaceCapability(ctx, caller, namespace, canManagePermissions); err != nil {
		if errors.Is(err, errForbidden) || errors.Is(err, errUnauthorized) {
			return "", err
		}
		if errors.Is(err, backend.ErrNamespaceNotFound) {
			return api.NamespaceNotFound, err
		}
		return api.DatabaseError, err
	}
	if username == caller.Username && !caller.Superuser {
		return "", errForbidden
	}

	if err := s.store.DeleteNamespacePermission(ctx, namespace, username); err != nil {
		if specError, ok := mapAuthorizationBackendError(err); ok {
			return specError, err
		}
		return api.DatabaseError, err
	}
	return "", nil
}
