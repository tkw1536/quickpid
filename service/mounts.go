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

// GetMount resolves a base URI to a namespace mount.
//
// It can return the following errors:
//
// - [api.MountNotFound]
// - [api.DatabaseError].
func (s *Service) GetMount(ctx context.Context, caller *api.Caller, baseURI api.ValidBaseURI) (*api.MountResponse, error) {
	namespaceID, err := s.backend.GetMount(ctx, baseURI)
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

// ListMounts lists namespace mounts.
//
// In anonymous mode, or when the caller is a superuser, all mounts are returned.
// Otherwise only mounts whose base URI matches at least one of the caller's mount grants are returned.
//
// It can return the following errors:
//
// - [api.DatabaseError].
func (s *Service) ListMounts(ctx context.Context, caller *api.Caller, params api.ListMountsParams) (*api.PaginatedMountsResponse, error) {
	// Optimization: If we have a superuser, we don't need to filter by mount grants.
	if caller == nil || caller.Superuser() {
		out, err := s.backend.ListMounts(ctx, params)
		if err != nil {
			return nil, api.WithErrorCode(fmt.Errorf("backend failed to list mounts: %w", err), api.DatabaseError)
		}
		return out, nil
	}

	grants, err := s.getAllCompiledMounts(ctx, caller.Username())
	if err != nil {
		return nil, err
	}

	pageLimit := s.Options().Limits.InternalPageLimit()

	filtered, err := filter.Filter(ctx, func(ctx context.Context, limit, offset int) ([]api.MountResponse, error) {
		page, err := s.backend.ListMounts(ctx, api.ListMountsParams{
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			return nil, api.WithErrorCode(fmt.Errorf("backend failed to list mounts: %w", err), api.DatabaseError)
		}
		return page.Items, nil
	}, func(ctx context.Context, mount api.MountResponse) (bool, error) {
		for _, grant := range grants {
			if grant.MatchString(mount.BaseURI) {
				return true, nil
			}
		}
		return false, nil
	}, pageLimit, params.Limit, params.Offset)

	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("filter returned error: %w", err), api.DatabaseError)
	}
	items := filtered.Items
	if items == nil {
		items = []api.MountResponse{}
	}
	return &api.PaginatedMountsResponse{
		Total:  filtered.Total,
		Offset: filtered.Offset,
		Items:  items,
	}, nil
}

func (s *Service) getAllCompiledMounts(loadCtx context.Context, username api.ValidUsername) ([]api.ValidMountGrantRequest, error) {
	loadCtx, cancel := context.WithCancel(loadCtx)
	defer cancel()

	grants := make([]api.ValidMountGrantRequest, 0, 100)
	for res := range s.loadCompiledMountGrants(loadCtx, username) {
		if res.err != nil {
			return nil, res.err
		}
		grants = append(grants, res.request)
	}
	return grants, nil
}

// ListNamespaceMounts lists base URIs mounted to a namespace.
//
// In authenticated mode, the caller must have at least the contributor role for the namespace.
// In anonymous mode, no authentication is required.
//
// It can return the following errors:
//
// - [api.NamespaceNotFound]
// - [api.DatabaseError].
func (s *Service) ListNamespaceMounts(ctx context.Context, caller *api.Caller, namespace api.ValidNamespaceID, params api.ListNamespaceMountsParams) (*api.PaginatedBaseURIResponse, error) {
	out, err := s.backend.ListNamespaceMounts(ctx, namespace, params)
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
// In authenticated mode, the caller must be a superuser or hold a matching mount grant.
// In anonymous mode, no authentication is required.
//
// It can return the following errors:
//
// - [api.NamespaceNotFound]
// - [api.DatabaseError].
func (s *Service) SetMount(ctx context.Context, caller *api.Caller, baseURI api.ValidBaseURI, req api.ValidMountUpsertRequest) (*api.MountResponse, error) {
	if err := s.backend.SetMount(ctx, baseURI, req.Namespace); err != nil {
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
// In authenticated mode, the caller must be a superuser or hold a matching mount grant.
// In anonymous mode, no authentication is required.
//
// It can return the following errors:
//
// - [api.MountNotFound]
// - [api.DatabaseError].
func (s *Service) DeleteMount(ctx context.Context, caller *api.Caller, baseURI api.ValidBaseURI) error {
	if err := s.backend.DeleteMount(ctx, baseURI); err != nil {
		if errors.Is(err, backend.ErrMountNotFound) {
			return api.WithErrorCode(fmt.Errorf("mount not found: %w", err), api.MountNotFound)
		}
		return api.WithErrorCode(fmt.Errorf("backend failed to delete mount: %w", err), api.DatabaseError)
	}
	return nil
}
