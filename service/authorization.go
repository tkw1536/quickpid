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

// requireAuthEnabled reports whether authentication features are available.
//
// It can return the following errors:
//
// - [api.UnavailableInAnonymousMode].
func (s *Service) requireAuthEnabled() error {
	if s.AnonymousMode() {
		return api.WithErrorString(errUnavailableInAnonymousMode, api.UnavailableInAnonymousMode)
	}
	return nil
}

// requireAuthenticated reports whether caller is authenticated.
//
// It can return the following errors:
//
// - [api.Unauthorized].
func (s *Service) requireAuthenticated(caller *api.ValidUserInfo) error {
	if caller == nil {
		return api.WithErrorString(errUnauthorized, api.Unauthorized)
	}
	return nil
}

// effectivePermission returns the caller's permission level in namespace.
//
// It can return the following errors:
//
// - [api.NamespaceNotFound]
// - [api.DatabaseError].
func (s *Service) effectivePermission(ctx context.Context, caller *api.ValidUserInfo, namespace *api.ValidNamespaceID) (api.PermissionLevel, error) {
	if caller.Superuser {
		return api.PermissionLevelManager, nil
	}

	level, err := s.store.GetNamespacePermission(ctx, namespace, caller.Username)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return level, api.WithErrorString(fmt.Errorf("Store.GetNamespacePermission: %w", err), api.NamespaceNotFound)
	}
	if err != nil {
		return level, api.WithErrorString(fmt.Errorf("Store.GetNamespacePermission: %w", err), api.DatabaseError)
	}
	return level, nil
}

// requireNamespaceCapability checks that caller is authenticated and has the given capability in namespace.
//
// It can return the following errors:
//
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.NamespaceNotFound]
// - [api.DatabaseError].
func (s *Service) requireNamespaceCapability(ctx context.Context, caller *api.ValidUserInfo, namespace *api.ValidNamespaceID, allowed func(api.PermissionLevel) bool) error {
	if err := s.requireAuthenticated(caller); err != nil {
		return err
	}
	level, err := s.effectivePermission(ctx, caller, namespace)
	if err != nil {
		return err
	}
	if !allowed(level) {
		return api.WithErrorString(errForbidden, api.Forbidden)
	}
	return nil
}

// mapAuthorizationBackendError translates backend store errors to API-annotated errors.
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
//
// It can return the following errors:
//
// - [api.UnavailableInAnonymousMode]
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.NamespaceNotFound]
// - [api.DatabaseError].
func (s *Service) GetNamespacePermission(ctx context.Context, caller *api.ValidUserInfo, namespace *api.ValidNamespaceID, username *api.ValidUsername) (*api.NamespacePermission, error) {
	if err := s.requireAuthEnabled(); err != nil {
		return nil, err
	}
	if err := s.requireAuthenticated(caller); err != nil {
		return nil, err
	}

	if username.String() != caller.Username.String() {
		if err := s.requireNamespaceCapability(ctx, caller, namespace, canManagePermissions); err != nil {
			return nil, err
		}
	}

	level, err := s.store.GetNamespacePermission(ctx, namespace, username)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.WithErrorString(fmt.Errorf("Store.GetNamespacePermission: %w", err), api.NamespaceNotFound)
	}
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("Store.GetNamespacePermission: %w", err), api.DatabaseError)
	}
	return &api.NamespacePermission{Username: username.String(), Level: level}, nil
}

// ListNamespacePermissions lists explicit permissions in a namespace. Caller must be a manager.
//
// It can return the following errors:
//
// - [api.UnavailableInAnonymousMode]
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.NamespaceNotFound]
// - [api.DatabaseError].
func (s *Service) ListNamespacePermissions(ctx context.Context, caller *api.ValidUserInfo, namespace *api.ValidNamespaceID, params api.ListNamespacePermissionsParams) (*api.PaginatedNamespacePermissionsResponse, error) {
	if err := s.requireAuthEnabled(); err != nil {
		return nil, err
	}
	if err := s.requireNamespaceCapability(ctx, caller, namespace, canManagePermissions); err != nil {
		return nil, err
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
//
// It can return the following errors:
//
// - [api.UnavailableInAnonymousMode]
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.InvalidPermissionLevel]
// - [api.NamespaceNotFound]
// - [api.DatabaseError].
func (s *Service) SetNamespacePermission(ctx context.Context, caller *api.ValidUserInfo, namespace *api.ValidNamespaceID, username *api.ValidUsername, req api.SetNamespacePermissionRequest) (*api.NamespacePermission, error) {
	if err := s.requireAuthEnabled(); err != nil {
		return nil, err
	}
	if err := s.requireNamespaceCapability(ctx, caller, namespace, canManagePermissions); err != nil {
		return nil, err
	}
	if username.String() == caller.Username.String() && !caller.Superuser {
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
	return &api.NamespacePermission{Username: username.String(), Level: level}, nil
}

// DeleteNamespacePermission removes an explicit permission record. Caller must be a manager.
//
// It can return the following errors:
//
// - [api.UnavailableInAnonymousMode]
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.NamespaceNotFound]
// - [api.PermissionNotFound]
// - [api.DatabaseError].
func (s *Service) DeleteNamespacePermission(ctx context.Context, caller *api.ValidUserInfo, namespace *api.ValidNamespaceID, username *api.ValidUsername) error {
	if err := s.requireAuthEnabled(); err != nil {
		return err
	}
	if _, err := s.store.GetNamespace(ctx, namespace); errors.Is(err, backend.ErrNamespaceNotFound) {
		return api.WithErrorString(fmt.Errorf("Store.GetNamespace: %w", err), api.NamespaceNotFound)
	} else if err != nil {
		return api.WithErrorString(fmt.Errorf("Store.GetNamespace: %w", err), api.DatabaseError)
	}
	if err := s.requireNamespaceCapability(ctx, caller, namespace, canManagePermissions); err != nil {
		return err
	}
	if username.String() == caller.Username.String() && !caller.Superuser {
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
