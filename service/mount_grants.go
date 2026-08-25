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

// ListUserMountGrants lists mount grant patterns for the caller.
//
// It can return the following errors:
//
// - [api.DatabaseError].
func (s *Service) ListUserMountGrants(ctx context.Context, caller *api.Caller, params api.ListUserMountGrantsParams) (*api.PaginatedMountGrantsResponse, error) {
	page, err := s.backend.ListUserMountGrants(ctx, caller.Username(), params)
	if errors.Is(err, backend.ErrUserNotFound) {
		// This SHOULD NEVER happen as we received the username from a user object.
		return nil, api.WithErrorCode(fmt.Errorf("%w: %w", errExistingUserNotFound, err), api.DatabaseError)
	}
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to list mount grants: %w", err), api.DatabaseError)
	}
	return page, nil
}

// SetMountGrant creates or replaces a mount grant for a user.
//
// It can return the following errors:
//
// - [api.UserNotFound]
// - [api.DatabaseError].
func (s *Service) SetMountGrant(ctx context.Context, caller *api.Caller, username api.ValidUsername, req api.ValidMountGrantRequest) (*api.MountGrant, error) {
	if err := s.backend.SetMountGrant(ctx, username, req); err != nil {
		if errors.Is(err, backend.ErrUserNotFound) {
			return nil, api.WithErrorCode(fmt.Errorf("user not found: %w", err), api.UserNotFound)
		}
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to set mount grant: %w", err), api.DatabaseError)
	}
	return &api.MountGrant{Pattern: req.String()}, nil
}

// DeleteMountGrant removes a mount grant for a user.
//
// It can return the following errors:
//
// - [api.UserNotFound]
// - [api.MountGrantNotFound]
// - [api.DatabaseError].
func (s *Service) DeleteMountGrant(ctx context.Context, caller *api.Caller, username api.ValidUsername, req api.ValidMountGrantRequest) error {
	if err := s.backend.DeleteMountGrant(ctx, username, req); err != nil {
		if errors.Is(err, backend.ErrUserNotFound) {
			return api.WithErrorCode(fmt.Errorf("user not found: %w", err), api.UserNotFound)
		}
		if errors.Is(err, backend.ErrMountGrantNotFound) {
			return api.WithErrorCode(fmt.Errorf("mount grant not found: %w", err), api.MountGrantNotFound)
		}
		return api.WithErrorCode(fmt.Errorf("backend failed to delete mount grant: %w", err), api.DatabaseError)
	}
	return nil
}

// MountGrantMatches reports whether any of the user's mount grants match the given base URI.
//
// It can return the following errors:
//
// - [api.DatabaseError].
func (s *Service) MountGrantMatches(loadCtx context.Context, username api.ValidUsername, baseURI api.ValidBaseURI) (bool, error) {
	loadCtx, cancel := context.WithCancel(loadCtx)
	defer cancel()

	target := baseURI.String()
	for res := range s.loadCompiledMountGrants(loadCtx, username) {
		if res.err != nil {
			return false, res.err
		}
		if res.request.MatchString(target) {
			return true, nil
		}
	}
	return false, nil
}

type mountGrantResult struct {
	err     error
	request api.ValidMountGrantRequest
}

// loadCompiledMountGrants loads and compiles all mount grants for a user.
// If an error occurs, the error is sent to the channel and the channel is closed.
func (s *Service) loadCompiledMountGrants(ctx context.Context, username api.ValidUsername) <-chan mountGrantResult {
	grants := make(chan mountGrantResult)
	go func() {
		defer close(grants)

		// lazily send a results to the channel
		send := func(result mountGrantResult) bool {
			select {
			case <-ctx.Done():
				return false
			case grants <- result:
				return true
			}
		}

		pageLimit := s.Options().Limits.InternalPageLimit()

		offset := 0
		for ctx.Err() == nil {
			page, err := s.backend.ListUserMountGrants(ctx, username, api.ListUserMountGrantsParams{
				Limit:  pageLimit,
				Offset: offset,
			})
			if errors.Is(err, backend.ErrUserNotFound) {
				send(mountGrantResult{err: api.WithErrorCode(fmt.Errorf("%w: %w", errExistingUserNotFound, err), api.DatabaseError)})
				return
			}
			if err != nil {
				send(mountGrantResult{err: api.WithErrorCode(fmt.Errorf("backend failed to list mount grants: %w", err), api.DatabaseError)})
				return
			}
			for _, item := range page.Items {
				compiled, err := api.NewMountGrantPattern(item.Pattern)
				if err != nil {
					send(mountGrantResult{err: api.WithErrorCode(fmt.Errorf("backend returned invalid mount grant pattern: %w", err), api.DatabaseError)})
					return
				}
				if !send(mountGrantResult{request: compiled}) {
					return
				}
			}
			offset += len(page.Items)
			if offset >= page.Total || len(page.Items) == 0 {
				break
			}
		}
	}()
	return grants
}
