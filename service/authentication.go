//spellchecker:words service
package service

//spellchecker:words context errors slog sync github quickpid backend internal apikey scopes
import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/internal/apikey"
	"github.com/tkw1536/quickpid/scopes"
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

// LoadUser loads and validates the user account for username.
//
// It can return the following errors:
//
// - [api.UserNotFound]
// - [api.DatabaseError].
func (s *Service) LoadUser(ctx context.Context, username api.ValidUsername) (api.ValidUserInfo, error) {
	user, err := s.store.GetUser(ctx, username)
	if errors.Is(err, backend.ErrUserNotFound) {
		return api.ValidUserInfo{}, api.WithErrorCode(fmt.Errorf("user not found: %w", err), api.UserNotFound)
	}
	if err != nil {
		return api.ValidUserInfo{}, api.WithErrorCode(fmt.Errorf("backend failed to get user: %w", err), api.DatabaseError)
	}
	info, err := user.Validate()
	if err != nil {
		return api.ValidUserInfo{}, api.WithErrorCode(fmt.Errorf("backend returned invalid user: %w", err), api.DatabaseError)
	}
	return info, nil
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
// - [api.DatabaseError]
// - [api.Forbidden].
func (s *Service) SetUserPassword(ctx context.Context, caller api.Caller, req api.ValidSetPasswordRequest) (*api.SetPasswordResponse, error) {
	if err := s.checkUserScope(&caller, scopes.ScopeSetPassword); err != nil {
		return nil, fmt.Errorf("SetPassword() check failed: %w", err)
	}

	hasPassword, err := s.store.SetPassword(ctx, caller.Username(), req.Password)
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to set password: %w", err), api.DatabaseError)
	}
	return &api.SetPasswordResponse{Password: hasPassword}, nil
}

// CreateUser creates a new user account. Caller must be a superuser.
//
// It can return the following errors:
//
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.DuplicateUsername]
// - [api.DatabaseError].
func (s *Service) CreateUser(ctx context.Context, caller api.Caller, req api.ValidUserCreateRequest) (*api.UserInfo, error) {
	if err := s.checkUserScope(&caller, scopes.ScopeCreateUser); err != nil {
		return nil, fmt.Errorf("CreateUser() check failed: %w", err)
	}
	user, err := s.store.CreateUser(ctx, req, s.runtime.Now)
	if mapped, ok := mapAuthBackendError(err); ok {
		return nil, mapped
	}
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to create user: %w", err), api.DatabaseError)
	}
	return user, nil
}

// DeleteUser deletes a user account. Caller must be a superuser and cannot delete their own account.
//
// It can return the following errors:
//
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.UserNotFound]
// - [api.DatabaseError].
func (s *Service) DeleteUser(ctx context.Context, caller api.Caller, target api.ValidUsername) error {
	if err := s.checkUserScope(&caller, scopes.ScopeDeleteUser); err != nil {
		return fmt.Errorf("DeleteUser() check failed: %w", err)
	}

	if target.String() == caller.Username().String() {
		return api.WithErrorCode(fmt.Errorf("cannot delete own account: %w", errForbidden), api.Forbidden)
	}

	err := s.store.DeleteUser(ctx, target)
	if mapped, ok := mapAuthBackendError(err); ok {
		return mapped
	}
	if err != nil {
		return api.WithErrorCode(fmt.Errorf("backend failed to delete user: %w", err), api.DatabaseError)
	}
	return nil
}

// ListUsers lists all user accounts. Caller must be a superuser.
//
// It can return the following errors:
//
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.DatabaseError].
func (s *Service) ListUsers(ctx context.Context, caller api.Caller, params api.ListUsersParams) (*api.PaginatedUsersResponse, error) {
	if err := s.checkUserScope(&caller, scopes.ScopeListUsers); err != nil {
		return nil, fmt.Errorf("ListUsers() check failed: %w", err)
	}

	page, err := s.store.ListUsers(ctx, params)
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to list users: %w", err), api.DatabaseError)
	}
	return page, nil
}

// AutocompleteUsers returns usernames matching a prefix. Any authenticated caller is allowed.
//
// It can return the following errors:
//
// - [api.DatabaseError].
func (s *Service) AutocompleteUsers(ctx context.Context, caller api.Caller, query api.ValidAutocompleteQuery) ([]string, error) {
	s.mu.RLock()
	limit := s.opts.Limits.MaxAutocompleteUsers
	s.mu.RUnlock()

	usernames, err := s.store.AutocompleteUsers(ctx, query, limit)
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to autocomplete users: %w", err), api.DatabaseError)
	}
	return usernames, nil
}

// ListKeys lists API keys for the caller.
//
// It can return the following errors:
//
// - [api.UnavailableInAnonymousMode]
// - [api.Forbidden]
// - [api.UserNotFound]
// - [api.DatabaseError].
func (s *Service) ListKeys(ctx context.Context, caller api.Caller, params api.ListKeysParams) (*api.PaginatedAPIKeysResponse, error) {
	if err := s.checkUserScope(&caller, scopes.ScopeListOwnKeys); err != nil {
		return nil, fmt.Errorf("ListOwnKeys() check failed: %w", err)
	}

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
	// The current api response
	//
	// The Total field will be updated as we encounter valid keys.
	var res api.PaginatedAPIKeysResponse

	// We want to check for all keys at a consistent time.
	// We also don't get the time if we don't need it.
	//
	// So we use [sync.OnceValue] for memoization
	// even though it will never be called concurrently.
	now := sync.OnceValue(s.runtime.Now)

	for key := range s.listAllKeys(ctx, format, username, params) {
		if key.err != nil {
			return nil, key.err
		}

		// skip invalid keys
		if !key.info.Valid(now) {
			continue
		}
		res.Total++

		// only add keys according to the underlying offset and limit.
		if res.Total > params.Offset && len(res.Items) < params.Limit {
			res.Items = append(res.Items, key.info)
		}
	}

	res.Offset = params.Offset
	return &res, nil
}

type keyInfoOrError struct {
	err  error
	info api.APIKeyInfo
}

// listAllKeys returns a channel that yields all keys for username.
//
// This ignores offset and limit of [ListKeysParams], but should respect any future parameters.
func (s *Service) listAllKeys(ctx context.Context, format apikey.Format, username api.ValidUsername, params api.ListKeysParams) chan keyInfoOrError {
	ch := make(chan keyInfoOrError)
	go func() {
		defer close(ch)

		// Determine the maximum page limit
		// for the configured size.
		limits := s.Options().Limits
		params.Limit = limits.MaxPageLimit
		if params.Limit < 1 {
			params.Limit = limits.DefaultPageLimit
		}

		params.Offset = 0

		for {
			// list the current page or bail out in case of an error.
			page, err := s.store.ListKeys(ctx, format, username, params)
			if err != nil {
				ch <- keyInfoOrError{err: err}
				return
			}

			// yield individual items.
			for _, info := range page.Items {
				ch <- keyInfoOrError{info: info}
			}

			// check if the next page still exists.
			params.Offset += len(page.Items)
			if len(page.Items) == 0 || params.Offset >= page.Total {
				return
			}
		}
	}()

	return ch
}

// IssueKey issues a new API key for caller or another user when caller is a superuser.
//
// It can return the following errors:
//
// - [api.UnavailableInAnonymousMode]
// - [api.InvalidQueryParameter]
// - [api.BadIDGeneration]
// - [api.DatabaseError]
// - [api.InsufficientEntropy]
// - [api.Forbidden].
func (s *Service) IssueKey(ctx context.Context, caller api.Caller, req api.KeyIssueRequest) (*api.IssueKeyResponse, error) {
	if err := s.checkUserScope(&caller, scopes.ScopeIssueKey); err != nil {
		return nil, fmt.Errorf("IssueKey() check failed: %w", err)
	}

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
	if err := s.checkUserScope(&caller, scopes.ScopeRevokeKey); err != nil {
		return fmt.Errorf("RevokeKey() check failed: %w", err)
	}

	err := s.store.RevokeKey(ctx, s.apiKeyFormat(), caller.Username(), req.ID)
	if errors.Is(err, backend.ErrKeyNotFound) {
		return api.WithErrorCode(fmt.Errorf("key not found: %w", err), api.KeyNotFound)
	}
	if err != nil {
		return api.WithErrorCode(fmt.Errorf("backend failed to revoke key: %w", err), api.DatabaseError)
	}
	return nil
}

// UpdateUser updates a user account. Caller must be a superuser and cannot update their own account.
//
// It can return the following errors:
//
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.UserNotFound]
// - [api.DatabaseError].
func (s *Service) UpdateUser(ctx context.Context, caller api.Caller, target api.ValidUsername, req api.UserUpdateRequest) (*api.UserInfo, error) {
	if err := s.checkUserScope(&caller, scopes.ScopeUpdateUser); err != nil {
		return nil, fmt.Errorf("UpdateUser() check failed: %w", err)
	}

	if target.String() == caller.Username().String() {
		return nil, api.WithErrorCode(errForbidden, api.Forbidden)
	}

	user, err := s.store.UpdateUser(ctx, target, req)
	if mapped, ok := mapAuthBackendError(err); ok {
		return nil, mapped
	}
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to update user: %w", err), api.DatabaseError)
	}
	return user, nil
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

var rootUsername api.ValidUsername

func init() {
	root, err := api.NewUsername("root")
	if err != nil {
		panic("never reached: root is a valid username")
	}
	rootUsername = root
}

// EnsureRootUser ensures that there is a user named "root" in the system.
// If there is no such user, it creates a new root user with superuser privileges, and generates and logs out an API key.
func (s *Service) EnsureRootUser(ctx context.Context, logger *slog.Logger) error {
	if s.AnonymousMode() {
		return nil
	}
	page, err := s.store.ListUsers(ctx, api.ListUsersParams{Limit: 1})
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}
	if page.Total > 0 {
		return nil
	}

	_, err = s.store.CreateUser(ctx, api.ValidUserCreateRequest{
		Username:  rootUsername,
		Superuser: true,
	}, s.runtime.Now)
	if errors.Is(err, backend.ErrDuplicateUsername) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to create root superuser: %w", err)
	}

	issued, err := s.issueAPIKey(ctx, rootUsername, api.KeyIssueRequest{
		Comment: "bootstrap",
	})
	if err != nil {
		return err
	}

	logger.Warn(
		"created root superuser with bootstrap API key",
		slog.String("username", rootUsername.String()),
		slog.String("key", issued.Key),
	)
	return nil
}

var (
	errCreateSuperUserAnonymousMode = errors.New("cannot create superuser in anonymous mode")
)

// CreateSuperuser creates a new superuser account with the given username, generates a new api key without expiry, and logs out the key.
//
// Other failures are returned as plain (unannotated) errors.
func (s *Service) CreateSuperuser(ctx context.Context, logger *slog.Logger, username string) error {
	if s.AnonymousMode() {
		return errCreateSuperUserAnonymousMode
	}

	validUsername, err := api.NewUsername(username)
	if err != nil {
		return fmt.Errorf("failed to parse username: %w", err)
	}

	_, err = s.store.CreateUser(ctx, api.ValidUserCreateRequest{
		Username:  validUsername,
		Superuser: true,
	}, s.runtime.Now)
	if err != nil {
		return fmt.Errorf("failed to create superuser: %w", err)
	}

	issued, err := s.issueAPIKey(ctx, validUsername, api.KeyIssueRequest{
		Comment: "bootstrap",
	})
	if err != nil {
		return fmt.Errorf("failed to issue api key: %w", err)
	}

	logger.Warn(
		"created superuser with API key",
		slog.String("username", validUsername.String()),
		slog.String("key", issued.Key),
	)
	return nil
}
