//spellchecker:words service
package service

//spellchecker:words context errors sync github quickpid backend internal apikey filter
import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/internal/apikey"
	"github.com/tkw1536/quickpid/internal/filter"
)

var (
	errUnauthorized    = errors.New("unauthorized")
	errExpiresAtInPast = errors.New("expires_at is in the past")
)

// AuthenticateAPIKey looks up the username for a valid API key.
func (s *Service) AuthenticateAPIKey(ctx context.Context, apiKey string) (api.ValidUsername, api.APIKeyInfo, error) {
	if s.AnonymousMode() || apiKey == "" {
		return api.ValidUsername{}, api.APIKeyInfo{}, errUnauthorized
	}

	username, key, err := s.store.LookupUserByKey(ctx, s.apiKeyFormat(), apiKey)
	if errors.Is(err, backend.ErrInvalidKey) {
		return api.ValidUsername{}, api.APIKeyInfo{}, fmt.Errorf("database found no key associated with the given user: %w", err)
	}
	if err != nil {
		return api.ValidUsername{}, api.APIKeyInfo{}, fmt.Errorf("unknown error while looking up key: %w", err)
	}
	if !key.Valid(s.runtime.Now) {
		return api.ValidUsername{}, api.APIKeyInfo{}, fmt.Errorf("key is expired: %w", errUnauthorized)
	}

	name, err := api.NewUsername(username)
	if err != nil {
		return api.ValidUsername{}, api.APIKeyInfo{}, fmt.Errorf("backend returned invalid username: %w", err)
	}
	return name, *key, nil
}

// AuthenticatePassword authenticates a username/password pair and returns the username.
//
// May return errors that wrap [api.Unauthorized].
func (s *Service) AuthenticatePassword(ctx context.Context, username string, password string) (api.ValidUsername, error) {
	if s.AnonymousMode() || username == "" || password == "" {
		return api.ValidUsername{}, api.WithErrorCode(errUnauthorized, api.Unauthorized)
	}

	validUsername, err := api.NewUsername(username)
	if err != nil {
		return api.ValidUsername{}, api.WithErrorCode(fmt.Errorf("invalid username provided: %w", err), api.Unauthorized)
	}
	validPassword, err := api.NewPassword(password)
	if err != nil {
		return api.ValidUsername{}, api.WithErrorCode(fmt.Errorf("failed to parse username: %w", err), api.Unauthorized)
	}

	ok, err := s.store.CheckPassword(ctx, validUsername, validPassword)
	if errors.Is(err, backend.ErrUserNotFound) {
		return api.ValidUsername{}, api.WithErrorCode(fmt.Errorf("user not found: %w", err), api.Unauthorized)
	}
	if err != nil {
		return api.ValidUsername{}, fmt.Errorf("backend reported the username password combination is invalid: %w", err)
	}
	if !ok {
		return api.ValidUsername{}, api.WithErrorCode(errUnauthorized, api.Unauthorized)
	}
	return validUsername, nil
}

// CheckPassword reports whether password matches the stored password for username.
func (s *Service) CheckPassword(ctx context.Context, username api.ValidUsername, password api.ValidPassword) (bool, error) {
	ok, err := s.store.CheckPassword(ctx, username, password)
	if mapped, handled := mapAuthBackendError(err); handled {
		return false, mapped
	}
	if err != nil {
		return false, fmt.Errorf("backend reported the username password combination is invalid: %w", err)
	}
	return ok, nil
}

// SetUserPassword sets or clears a password for the caller.
//
// It can return the following errors:
//
// - [api.DatabaseError].
func (s *Service) SetUserPassword(ctx context.Context, caller api.Caller, req api.ValidSetPasswordRequest) (*api.SetPasswordResponse, error) {
	hasPassword, err := s.store.SetPassword(ctx, caller.Username(), req.Password)
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to set password: %w", err), api.DatabaseError)
	}
	return &api.SetPasswordResponse{Password: hasPassword}, nil
}

// ListKeys lists API keys for the caller.
//
// It can return the following errors:
//
// - [api.UserNotFound]
// - [api.DatabaseError].
func (s *Service) ListKeys(ctx context.Context, caller api.Caller, params api.ListKeysParams) (*api.PaginatedAPIKeysResponse, error) {
	page, err := s.listValidKeys(ctx, s.apiKeyFormat(), caller.Username(), params)
	if mapped, ok := mapAuthBackendError(err); ok {
		return nil, mapped
	}
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to list keys: %w", err), api.DatabaseError)
	}
	return page, nil
}

// listValidKeys is like [s.store.ListKeys], but only contains keys that are still valid according to the runtime.
func (s *Service) listValidKeys(ctx context.Context, format apikey.Format, username api.ValidUsername, params api.ListKeysParams) (*api.PaginatedAPIKeysResponse, error) {
	// Determine the effective page limit to call the backend with.
	options := s.Options().Limits
	pageLimit := options.DefaultPageLimit
	if options.MaxPageLimit > 0 && pageLimit > options.MaxPageLimit {
		pageLimit = options.MaxPageLimit
	}

	// We want to check for all keys at a consistent time.
	// We also don't get the time if we don't need it.
	//
	// So we use [sync.OnceValue] for memoization
	// even though it will never be called concurrently.
	now := sync.OnceValue(s.runtime.Now)

	// Filter the keys to only includes ones that are still valid according to the current time.
	// This is a client-side filter, because the store backend does not have access to the runtime,
	// and even then is not expected to be able to reduce this to a database query.
	keys, err := filter.Filter(
		ctx,
		func(ctx context.Context, limit, offset int) ([]api.APIKeyInfo, error) {
			// Create new params for each page by shadowing the original params.
			// We could create a new struct, but this includes any future parameters for listing.
			params := params
			params.Limit = limit
			params.Offset = offset

			// List the actual keys, and return only the items.
			result, err := s.store.ListKeys(ctx, format, username, params)
			if err != nil {
				return nil, fmt.Errorf("backend failed to list keys: %w", err)
			}
			return result.Items, nil
		},
		func(ctx context.Context, key api.APIKeyInfo) (bool, error) {
			if err := ctx.Err(); err != nil {
				return false, fmt.Errorf("context cancelled: %w", err)
			}
			return key.Valid(now), nil
		},
		pageLimit,
		params.Limit,
		params.Offset,
	)

	if err != nil {
		return nil, fmt.Errorf("filter returned error: %w", err)
	}
	return &api.PaginatedAPIKeysResponse{
		Items:  keys.Items,
		Total:  keys.Total,
		Offset: keys.Offset,
	}, nil
}

// IssueKey issues a new API key for caller or another user when caller is a superuser.
//
// It can return the following errors:
//
// - [api.InvalidQueryParameter]
// - [api.BadIDGeneration]
// - [api.DatabaseError]
// - [api.InsufficientEntropy].
func (s *Service) IssueKey(ctx context.Context, caller api.Caller, req api.KeyIssueRequest) (*api.IssueKeyResponse, error) {
	if req.ExpiresAt != nil {
		if !req.ExpiresAt.After(s.runtime.Now()) {
			return nil, api.WithErrorCode(errExpiresAtInPast, api.InvalidQueryParameter)
		}
	}

	issued, err := s.issueAPIKey(ctx, caller.Username(), req)
	if err != nil {
		return nil, err
	}
	return issued, nil
}

// issueAPIKey issues a new API key, retrying when storage reports [backend.ErrKeyCollision].
//
// It can return the following errors:
//
// - [api.UserNotFound]
// - [api.BadIDGeneration]
// - [api.DatabaseError]
// - [api.InsufficientEntropy].
func (s *Service) issueAPIKey(ctx context.Context, username api.ValidUsername, req api.KeyIssueRequest) (*api.IssueKeyResponse, error) {
	s.mu.RLock()
	maxAttempts := s.opts.Limits.MaxAPIKeyAttempts
	s.mu.RUnlock()

	format := s.apiKeyFormat()
	for range maxAttempts {
		keyID, err := s.runtime.NewAPIKeyID()
		if err != nil {
			return nil, api.WithErrorCode(fmt.Errorf("failed to generate new api key id: %w", err), api.BadIDGeneration)
		}

		rawKey, err := s.runtime.NewAPIKey(format)
		if err != nil {
			return nil, api.WithErrorCode(fmt.Errorf("failed to generate new api key: %w", err), api.BadIDGeneration)
		}

		info, err := s.store.CreateKey(ctx, format, username, keyID, rawKey, req, s.runtime.Now)
		if err == nil {
			return &api.IssueKeyResponse{APIKeyInfo: *info, Key: rawKey}, nil
		}
		if mapped, ok := mapAuthBackendError(err); ok {
			return nil, mapped
		}
		if !errors.Is(err, backend.ErrKeyCollision) {
			return nil, api.WithErrorCode(fmt.Errorf("backend failed to create key: %w", err), api.DatabaseError)
		}
	}
	return nil, api.WithErrorCode(fmt.Errorf("%w: gave up api key generation after %d attempts", errInsufficientEntropy, maxAttempts), api.InsufficientEntropy)
}

// RevokeKey revokes an API key for the caller or another user when caller is a superuser.
//
// It can return the following errors:
//
// - [api.KeyNotFound]
// - [api.DatabaseError].
func (s *Service) RevokeKey(ctx context.Context, caller api.Caller, req api.KeyRevokeRequest) error {
	err := s.store.RevokeKey(ctx, s.apiKeyFormat(), caller.Username(), req.ID)
	if errors.Is(err, backend.ErrKeyNotFound) {
		return api.WithErrorCode(fmt.Errorf("key not found: %w", err), api.KeyNotFound)
	}
	if err != nil {
		return api.WithErrorCode(fmt.Errorf("backend failed to revoke key: %w", err), api.DatabaseError)
	}
	return nil
}

func (s *Service) apiKeyFormat() apikey.Format {
	return apikey.Default
}

// mapAuthBackendError translates backend store errors to API-annotated errors.
func mapAuthBackendError(err error) (error, bool) {
	switch {
	case errors.Is(err, backend.ErrDuplicateUsername):
		return api.WithErrorCode(fmt.Errorf("duplicate username: %w", err), api.DuplicateUsername), true
	case errors.Is(err, backend.ErrUserNotFound):
		return api.WithErrorCode(fmt.Errorf("user not found: %w", err), api.UserNotFound), true
	case errors.Is(err, backend.ErrKeyNotFound):
		return api.WithErrorCode(fmt.Errorf("key not found: %w", err), api.KeyNotFound), true
	default:
		return nil, false
	}
}
