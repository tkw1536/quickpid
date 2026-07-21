//spellchecker:words service
package service

//spellchecker:words context errors slog time github quickpid backend internal apikey
import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/internal/apikey"
)

// AuthenticateAPIKey looks up the username for a valid API key.
//
// It can return the following errors:
//
// - [errUnauthorized] when the key is missing or invalid.
//
// Other failures are returned as plain (unannotated) errors.
func (s *Service) AuthenticateAPIKey(ctx context.Context, apiKey string) (api.ValidUsername, error) {
	if s.AnonymousMode() || apiKey == "" {
		return api.ValidUsername{}, errUnauthorized
	}

	username, err := s.store.LookupUserByKey(ctx, s.apiKeyFormat(), apiKey)
	if errors.Is(err, backend.ErrInvalidKey) {
		return api.ValidUsername{}, errUnauthorized
	}
	if err != nil {
		return api.ValidUsername{}, fmt.Errorf("store.LookupUserByKey: %w", err)
	}

	name, err := api.NewUsername(username)
	if err != nil {
		return api.ValidUsername{}, fmt.Errorf("api.NewUsername: %w", err)
	}
	return name, nil
}

// AuthenticatePassword authenticates a username/password pair and returns the username.
//
// It can return the following errors:
//
//   - [errUnauthorized] when authentication is disabled, credentials are missing or invalid,
//     or the user does not exist.
//
// Other failures are returned as plain (unannotated) errors.
func (s *Service) AuthenticatePassword(ctx context.Context, username string, password string) (api.ValidUsername, error) {
	if s.AnonymousMode() || username == "" || password == "" {
		return api.ValidUsername{}, errUnauthorized
	}

	validUsername, err := api.NewUsername(username)
	if err != nil {
		return api.ValidUsername{}, errUnauthorized
	}
	validPassword, err := api.NewPassword(password)
	if err != nil {
		return api.ValidUsername{}, errUnauthorized
	}

	ok, err := s.store.CheckPassword(ctx, validUsername, validPassword)
	if errors.Is(err, backend.ErrUserNotFound) {
		return api.ValidUsername{}, errUnauthorized
	}
	if err != nil {
		return api.ValidUsername{}, fmt.Errorf("store.CheckPassword: %w", err)
	}
	if !ok {
		return api.ValidUsername{}, errUnauthorized
	}
	return validUsername, nil
}

// LoadUser loads and validates the user account for username.
//
// It can return the following errors:
//
// - [errUnauthorized] when authentication is disabled or the user does not exist.
//
// Other failures are returned as plain (unannotated) errors.
func (s *Service) LoadUser(ctx context.Context, username api.ValidUsername) (api.ValidUserInfo, error) {
	if s.AnonymousMode() {
		return api.ValidUserInfo{}, errUnauthorized
	}

	user, err := s.store.GetUser(ctx, username)
	if errors.Is(err, backend.ErrUserNotFound) {
		return api.ValidUserInfo{}, errUnauthorized
	}
	if err != nil {
		return api.ValidUserInfo{}, fmt.Errorf("store.GetUser: %w", err)
	}
	info, err := user.Validate()
	if err != nil {
		return api.ValidUserInfo{}, fmt.Errorf("user.Validate: %w", err)
	}
	return info, nil
}

// GetUserAccount returns the caller's account or another user's when caller is a superuser.
//
// It can return the following errors:
//
// - [api.UnavailableInAnonymousMode]
// - [api.Forbidden]
// - [api.UserNotFound]
// - [api.DatabaseError].
func (s *Service) GetUserAccount(ctx context.Context, caller *api.ValidUserInfo, target *api.ValidUsername) (*api.UserInfo, error) {
	if err := s.requireAuthEnabled(); err != nil {
		return nil, err
	}
	username, err := resolveTarget(caller, target)
	if err != nil {
		return nil, err
	}
	user, err := s.store.GetUser(ctx, username)
	if mapped, ok := mapAuthBackendError(err); ok {
		return nil, fmt.Errorf("store.GetUser: %w", mapped)
	}
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("store.GetUser: %w", err), api.DatabaseError)
	}
	return user, nil
}

// GetUser returns the user account for caller.
//
// It can return the following errors:
//
// - [api.UnavailableInAnonymousMode]
// - [api.UserNotFound]
// - [api.DatabaseError].
func (s *Service) GetUser(ctx context.Context, caller *api.ValidUserInfo) (*api.UserInfo, error) {
	if err := s.requireAuthEnabled(); err != nil {
		return nil, err
	}
	user, err := s.store.GetUser(ctx, caller.Username)
	if mapped, ok := mapAuthBackendError(err); ok {
		return nil, fmt.Errorf("store.GetUser: %w", mapped)
	}
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("store.GetUser: %w", err), api.DatabaseError)
	}
	return user, nil
}

// SetUserPassword sets or clears a password for the caller or another user when caller is a superuser.
//
// It can return the following errors:
//
// - [api.UnavailableInAnonymousMode]
// - [api.Forbidden]
// - [api.UserNotFound]
// - [api.DatabaseError].
func (s *Service) SetUserPassword(ctx context.Context, caller *api.ValidUserInfo, target *api.ValidUsername, req api.ValidSetPasswordRequest) (*api.SetPasswordResponse, error) {
	if err := s.requireAuthEnabled(); err != nil {
		return nil, err
	}
	username, err := resolveTarget(caller, target)
	if err != nil {
		return nil, err
	}
	hasPassword, err := s.store.SetPassword(ctx, username, req.Password)
	if mapped, ok := mapAuthBackendError(err); ok {
		return nil, fmt.Errorf("store.SetPassword: %w", mapped)
	}
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("store.SetPassword: %w", err), api.DatabaseError)
	}
	return &api.SetPasswordResponse{Password: hasPassword}, nil
}

// CheckPassword reports whether password matches the stored password for username.
func (s *Service) CheckPassword(ctx context.Context, username api.ValidUsername, password api.ValidPassword) (bool, error) {
	ok, err := s.store.CheckPassword(ctx, username, password)
	if mapped, handled := mapAuthBackendError(err); handled {
		return false, fmt.Errorf("store.CheckPassword: %w", mapped)
	}
	if err != nil {
		return false, fmt.Errorf("store.CheckPassword: %w", err)
	}
	return ok, nil
}

// CreateUser creates a new user account. Caller must be a superuser.
//
// It can return the following errors:
//
// - [api.UnavailableInAnonymousMode]
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.DuplicateUsername]
// - [api.DatabaseError].
func (s *Service) CreateUser(ctx context.Context, caller *api.ValidUserInfo, req api.ValidUserCreateRequest) (*api.UserInfo, error) {
	if err := s.requireAuthEnabled(); err != nil {
		return nil, err
	}
	if err := requireSuperuser(caller); err != nil {
		return nil, err
	}
	if req.Superuser && !caller.Superuser {
		return nil, api.WithErrorString(errForbidden, api.Forbidden)
	}

	user, err := s.store.CreateUser(ctx, req, s.runtime.Now)
	if mapped, ok := mapAuthBackendError(err); ok {
		return nil, fmt.Errorf("store.CreateUser: %w", mapped)
	}
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("store.CreateUser: %w", err), api.DatabaseError)
	}
	return user, nil
}

// ListUsers lists all user accounts. Caller must be a superuser.
//
// It can return the following errors:
//
// - [api.UnavailableInAnonymousMode]
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.DatabaseError].
func (s *Service) ListUsers(ctx context.Context, caller *api.ValidUserInfo, params api.ListUsersParams) (*api.PaginatedUsersResponse, error) {
	if err := s.requireAuthEnabled(); err != nil {
		return nil, err
	}
	if err := requireSuperuser(caller); err != nil {
		return nil, err
	}

	page, err := s.store.ListUsers(ctx, params)
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("store.ListUsers: %w", err), api.DatabaseError)
	}
	return page, nil
}

// AutocompleteUsers returns usernames matching a prefix. Any authenticated caller is allowed.
//
// It can return the following errors:
//
// - [api.UnavailableInAnonymousMode]
// - [api.DatabaseError].
func (s *Service) AutocompleteUsers(ctx context.Context, caller *api.ValidUserInfo, query string) ([]string, error) {
	if err := s.requireAuthEnabled(); err != nil {
		return nil, err
	}
	_ = caller

	s.mu.RLock()
	limit := s.opts.Limits.MaxAutocompleteUsers
	s.mu.RUnlock()

	usernames, err := s.store.AutocompleteUsers(ctx, query, limit)
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("store.AutocompleteUsers: %w", err), api.DatabaseError)
	}
	return usernames, nil
}

// ListKeys lists API keys for the caller or another user when caller is a superuser.
//
// It can return the following errors:
//
// - [api.UnavailableInAnonymousMode]
// - [api.Forbidden]
// - [api.UserNotFound]
// - [api.DatabaseError].
func (s *Service) ListKeys(ctx context.Context, caller *api.ValidUserInfo, target *api.ValidUsername, params api.ListKeysParams) (*api.PaginatedAPIKeysResponse, error) {
	if err := s.requireAuthEnabled(); err != nil {
		return nil, err
	}
	username, err := resolveTarget(caller, target)
	if err != nil {
		return nil, err
	}
	page, err := s.store.ListKeys(ctx, s.apiKeyFormat(), username, params)
	if mapped, ok := mapAuthBackendError(err); ok {
		return nil, fmt.Errorf("store.ListKeys: %w", mapped)
	}
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("store.ListKeys: %w", err), api.DatabaseError)
	}
	return page, nil
}

// IssueKey issues a new API key for caller or another user when caller is a superuser.
//
// It can return the following errors:
//
// - [api.UnavailableInAnonymousMode]
// - [api.Forbidden]
// - [api.InvalidQueryParameter]
// - [api.UserNotFound]
// - [api.BadIDGeneration]
// - [api.DatabaseError]
// - [api.InsufficientEntropy].
func (s *Service) IssueKey(ctx context.Context, caller *api.ValidUserInfo, target *api.ValidUsername, req api.KeyIssueRequest) (*api.IssueKeyResponse, error) {
	if err := s.requireAuthEnabled(); err != nil {
		return nil, err
	}
	username, err := resolveTarget(caller, target)
	if err != nil {
		return nil, err
	}
	if req.ExpiresAt != nil {
		expiresAt, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			return nil, api.WithErrorString(fmt.Errorf("time.Parse: %w", err), api.InvalidQueryParameter)
		}
		if !expiresAt.After(s.runtime.Now()) {
			return nil, api.WithErrorString(errExpiresAtInPast, api.InvalidQueryParameter)
		}
	}

	if _, err := s.store.GetUser(ctx, username); err != nil {
		if mapped, ok := mapAuthBackendError(err); ok {
			return nil, fmt.Errorf("store.GetUser: %w", mapped)
		}
		return nil, api.WithErrorString(fmt.Errorf("store.GetUser: %w", err), api.DatabaseError)
	}

	issued, err := s.issueAPIKey(ctx, username, req)
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
			return nil, api.WithErrorString(fmt.Errorf("runtime.NewAPIKeyID: %w", err), api.BadIDGeneration)
		}

		rawKey, err := s.runtime.NewAPIKey(format)
		if err != nil {
			return nil, api.WithErrorString(fmt.Errorf("runtime.NewAPIKey: %w", err), api.BadIDGeneration)
		}

		info, err := s.store.CreateKey(ctx, format, username, keyID, rawKey, req, s.runtime.Now)
		if err == nil {
			return &api.IssueKeyResponse{APIKeyInfo: *info, Key: rawKey}, nil
		}
		if mapped, ok := mapAuthBackendError(err); ok {
			return nil, fmt.Errorf("store.CreateKey: %w", mapped)
		}
		if !errors.Is(err, backend.ErrKeyCollision) {
			return nil, api.WithErrorString(fmt.Errorf("store.CreateKey: %w", err), api.DatabaseError)
		}
	}
	return nil, api.WithErrorString(fmt.Errorf("%w: gave up api key generation after %d attempts", errInsufficientEntropy, maxAttempts), api.InsufficientEntropy)
}

// RevokeKey revokes an API key for the caller or another user when caller is a superuser.
//
// It can return the following errors:
//
// - [api.UnavailableInAnonymousMode]
// - [api.Forbidden]
// - [api.UserNotFound]
// - [api.KeyNotFound]
// - [api.DatabaseError].
func (s *Service) RevokeKey(ctx context.Context, caller *api.ValidUserInfo, target *api.ValidUsername, req api.KeyRevokeRequest) (*api.RevokeKeyResponse, error) {
	if err := s.requireAuthEnabled(); err != nil {
		return nil, err
	}
	username, err := resolveTarget(caller, target)
	if err != nil {
		return nil, err
	}
	info, err := s.store.RevokeKey(ctx, s.apiKeyFormat(), username, req.ID)
	if mapped, ok := mapAuthBackendError(err); ok {
		return nil, fmt.Errorf("store.RevokeKey: %w", mapped)
	}
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("store.RevokeKey: %w", err), api.DatabaseError)
	}
	return &api.RevokeKeyResponse{APIKeyInfo: *info}, nil
}

// UpdateUser updates a user account. Caller must be a superuser and cannot update their own account.
//
// It can return the following errors:
//
// - [api.UnavailableInAnonymousMode]
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.UserNotFound]
// - [api.DatabaseError].
func (s *Service) UpdateUser(ctx context.Context, caller *api.ValidUserInfo, target api.ValidUsername, req api.UserUpdateRequest) (*api.UserInfo, error) {
	if err := s.requireAuthEnabled(); err != nil {
		return nil, err
	}
	if err := requireSuperuser(caller); err != nil {
		return nil, err
	}

	if target.String() == caller.Username.String() {
		return nil, api.WithErrorString(errForbidden, api.Forbidden)
	}

	user, err := s.store.UpdateUser(ctx, target, req)
	if mapped, ok := mapAuthBackendError(err); ok {
		return nil, fmt.Errorf("store.UpdateUser: %w", mapped)
	}
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("store.UpdateUser: %w", err), api.DatabaseError)
	}
	return user, nil
}

// resolveTarget resolves a "target" username.
//
// When the caller is not a superuser, it only permits the caller's own username (or nil, which automatically defaults to the caller's username).
// Otherwise, it allows the target username.
//
// It can return the following errors:
//
// - [api.Forbidden].
func resolveTarget(caller *api.ValidUserInfo, target *api.ValidUsername) (api.ValidUsername, error) {
	// TODO: Are there more places where we can use this function?

	// no target, or target === caller
	if target == nil || target.String() == caller.Username.String() {
		return caller.Username, nil
	}

	if !caller.Superuser {
		return api.ValidUsername{}, api.WithErrorString(errForbidden, api.Forbidden)
	}
	return *target, nil
}

// requireSuperuser reports whether user is a superuser.
//
// It can return the following errors:
//
// - [api.Unauthorized]
// - [api.Forbidden].
func requireSuperuser(user *api.ValidUserInfo) error {
	if user == nil {
		return api.WithErrorString(errUnauthorized, api.Unauthorized)
	}
	if !user.Superuser {
		return api.WithErrorString(errForbidden, api.Forbidden)
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
		return api.WithErrorString(err, api.DuplicateUsername), true
	case errors.Is(err, backend.ErrUserNotFound):
		return api.WithErrorString(err, api.UserNotFound), true
	case errors.Is(err, backend.ErrKeyNotFound):
		return api.WithErrorString(err, api.KeyNotFound), true
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
//
// It can return the following errors:
//
// - [api.UserNotFound]
// - [api.BadIDGeneration]
// - [api.DatabaseError]
// - [api.InsufficientEntropy]
//
// Other failures are returned as plain (unannotated) errors.
func (s *Service) EnsureRootUser(ctx context.Context, logger *slog.Logger) error {
	if s.AnonymousMode() {
		return nil
	}
	page, err := s.store.ListUsers(ctx, api.ListUsersParams{Limit: 1})
	if err != nil {
		return fmt.Errorf("store.ListUsers: %w", err)
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
