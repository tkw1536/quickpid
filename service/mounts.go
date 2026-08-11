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

// GetMount resolves a base URI to a namespace mount.
//
// It can return the following errors:
//
// - [api.MountNotFound]
// - [api.DatabaseError].
func (s *Service) GetMount(ctx context.Context, baseURI api.ValidBaseURI) (*api.MountResponse, error) {
	namespaceID, err := s.store.GetMount(ctx, baseURI)
	if errors.Is(err, backend.ErrMountNotFound) {
		return nil, api.WithErrorCode(fmt.Errorf("mount not found: %w", err), api.MountNotFound)
	}
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to get mount: %w", err), api.DatabaseError)
	}
	return &api.MountResponse{
		BaseURI:   baseURI.String(),
		Namespace: namespaceID,
	}, nil
}

// ListMounts lists all namespace mounts.
//
// In authenticated mode, the caller must be a superuser.
// In anonymous mode, no authentication is required.
//
// It can return the following errors:
//
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.DatabaseError].
func (s *Service) ListMounts(ctx context.Context, caller *api.AuthenticatedUser, params api.ListMountsParams) (*api.PaginatedMountsResponse, error) {
	if !s.AnonymousMode() {
		if err := requireAuthenticated(caller); err != nil {
			return nil, err
		}
		if err := requireSuperuser(*caller); err != nil {
			return nil, err
		}
	}

	out, err := s.store.ListMounts(ctx, params)
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to list mounts: %w", err), api.DatabaseError)
	}
	return out, nil
}

// ListNamespaceMounts lists base URIs mounted to a namespace.
//
// In authenticated mode, the caller must have at least the contributor role for the namespace.
// In anonymous mode, no authentication is required.
//
// It can return the following errors:
//
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.NamespaceNotFound]
// - [api.DatabaseError].
func (s *Service) ListNamespaceMounts(ctx context.Context, caller *api.AuthenticatedUser, namespace api.ValidNamespaceID, params api.ListNamespaceMountsParams) (*api.PaginatedBaseURIResponse, error) {
	if !s.AnonymousMode() {
		if err := requireAuthenticated(caller); err != nil {
			return nil, err
		}
		if err := s.requireNamespaceCapability(ctx, *caller, namespace, canReadNamespaceMetadata); err != nil {
			return nil, err
		}
	}

	out, err := s.store.ListNamespaceMounts(ctx, namespace, params)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.WithErrorCode(fmt.Errorf("namespace not found: %w", err), api.NamespaceNotFound)
	}
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to list namespace mounts: %w", err), api.DatabaseError)
	}
	return out, nil
}

// SetMount creates or replaces a namespace mount.
//
// In authenticated mode, the caller must be a superuser.
// In anonymous mode, no authentication is required.
//
// It can return the following errors:
//
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.NamespaceNotFound]
// - [api.DatabaseError].
func (s *Service) SetMount(ctx context.Context, caller *api.AuthenticatedUser, baseURI api.ValidBaseURI, req api.ValidMountUpsertRequest) (*api.MountResponse, error) {
	if !s.AnonymousMode() {
		if err := requireAuthenticated(caller); err != nil {
			return nil, err
		}
		if err := requireSuperuser(*caller); err != nil {
			return nil, err
		}
	}

	if err := s.store.SetMount(ctx, baseURI, req.Namespace); err != nil {
		if errors.Is(err, backend.ErrNamespaceNotFound) {
			return nil, api.WithErrorCode(fmt.Errorf("namespace not found: %w", err), api.NamespaceNotFound)
		}
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to set mount: %w", err), api.DatabaseError)
	}

	return &api.MountResponse{
		BaseURI:   baseURI.String(),
		Namespace: req.Namespace.String(),
	}, nil
}

// DeleteMount removes a namespace mount.
//
// In authenticated mode, the caller must be a superuser.
// In anonymous mode, no authentication is required.
//
// It can return the following errors:
//
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.MountNotFound]
// - [api.DatabaseError].
func (s *Service) DeleteMount(ctx context.Context, caller *api.AuthenticatedUser, baseURI api.ValidBaseURI) error {
	if !s.AnonymousMode() {
		if err := requireAuthenticated(caller); err != nil {
			return err
		}
		if err := requireSuperuser(*caller); err != nil {
			return err
		}
	}

	if err := s.store.DeleteMount(ctx, baseURI); err != nil {
		if errors.Is(err, backend.ErrMountNotFound) {
			return api.WithErrorCode(fmt.Errorf("mount not found: %w", err), api.MountNotFound)
		}
		return api.WithErrorCode(fmt.Errorf("backend failed to delete mount: %w", err), api.DatabaseError)
	}
	return nil
}
