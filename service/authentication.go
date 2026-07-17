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

// Authenticate looks up the username for a valid API key.
// It returns errUnauthorized when the key is missing or invalid.
func (s *Service) Authenticate(ctx context.Context, apiKey string) (string, error) {
	if s.AnonymousMode() {
		return "", errUnauthorized
	}
	if apiKey == "" {
		return "", errUnauthorized
	}

	username, err := s.store.LookupUserByKey(ctx, s.apiKeyFormat(), apiKey)
	if errors.Is(err, backend.ErrInvalidKey) {
		return "", errUnauthorized
	}
	if err != nil {
		return "", fmt.Errorf("store.LookupUserByKey: %w", err)
	}
	return username, nil
}

// CurrentUser authenticates an API key and returns the corresponding user account.
// It returns errUnauthorized when the key is missing, invalid, or the user no longer exists.
func (s *Service) CurrentUser(ctx context.Context, apiKey string) (*api.UserInfo, error) {
	if s.AnonymousMode() {
		return nil, errUnauthorized
	}
	caller, err := s.Authenticate(ctx, apiKey)
	if err != nil {
		return nil, err
	}
	user, err := s.store.GetUser(ctx, caller)
	if errors.Is(err, backend.ErrUserNotFound) {
		return nil, errUnauthorized
	}
	if err != nil {
		return nil, fmt.Errorf("store.GetUser: %w", err)
	}
	return user, nil
}

// GetUserAccount returns the caller's account or another user's when caller is a superuser.
func (s *Service) GetUserAccount(ctx context.Context, caller *api.UserInfo, target *string) (*api.UserInfo, error) {
	if err := s.requireAuthEnabled(); err != nil {
		return nil, err
	}
	username, err := resolveTargetUsername(caller, target)
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
func (s *Service) GetUser(ctx context.Context, caller string) (*api.UserInfo, error) {
	if err := s.requireAuthEnabled(); err != nil {
		return nil, err
	}
	user, err := s.store.GetUser(ctx, caller)
	if mapped, ok := mapAuthBackendError(err); ok {
		return nil, fmt.Errorf("store.GetUser: %w", mapped)
	}
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("store.GetUser: %w", err), api.DatabaseError)
	}
	return user, nil
}

// CreateUser creates a new user account. Caller must be a superuser.
func (s *Service) CreateUser(ctx context.Context, caller *api.UserInfo, req api.UserCreateRequest) (*api.UserInfo, error) {
	if err := s.requireAuthEnabled(); err != nil {
		return nil, err
	}
	if err := requireSuperuser(caller); err != nil {
		return nil, err
	}
	if err := ValidateUsername(req.Username); err != nil {
		return nil, api.WithErrorString(err, api.InvalidUsername)
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
func (s *Service) ListUsers(ctx context.Context, caller *api.UserInfo, params api.ListUsersParams) (*api.PaginatedUsersResponse, error) {
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
func (s *Service) AutocompleteUsers(ctx context.Context, caller *api.UserInfo, query string) ([]string, error) {
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
func (s *Service) ListKeys(ctx context.Context, caller *api.UserInfo, target *string, params api.ListKeysParams) (*api.PaginatedAPIKeysResponse, error) {
	if err := s.requireAuthEnabled(); err != nil {
		return nil, err
	}
	username, err := resolveTargetUsername(caller, target)
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
// - [api.Forbidden]
// - [api.BadIDGeneration]
// - [api.DatabaseError]
// - [api.InsufficientEntropy].
func (s *Service) IssueKey(ctx context.Context, caller *api.UserInfo, target *string, req api.KeyIssueRequest) (*api.IssueKeyResponse, error) {
	if err := s.requireAuthEnabled(); err != nil {
		return nil, err
	}
	username, err := resolveTargetUsername(caller, target)
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
// - [api.BadIDGeneration]
// - [api.DatabaseError]
// - [api.InsufficientEntropy].
func (s *Service) issueAPIKey(ctx context.Context, username string, req api.KeyIssueRequest) (*api.IssueKeyResponse, error) {
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
func (s *Service) RevokeKey(ctx context.Context, caller *api.UserInfo, target *string, req api.KeyRevokeRequest) (*api.RevokeKeyResponse, error) {
	if err := s.requireAuthEnabled(); err != nil {
		return nil, err
	}
	username, err := resolveTargetUsername(caller, target)
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
func (s *Service) UpdateUser(ctx context.Context, caller *api.UserInfo, target string, req api.UserUpdateRequest) (*api.UserInfo, error) {
	if err := s.requireAuthEnabled(); err != nil {
		return nil, err
	}
	if err := requireSuperuser(caller); err != nil {
		return nil, err
	}
	if target == caller.Username {
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

func resolveTargetUsername(caller *api.UserInfo, target *string) (string, error) {
	if target == nil {
		return caller.Username, nil
	}
	if *target != caller.Username && !caller.Superuser {
		return "", api.WithErrorString(errForbidden, api.Forbidden)
	}
	return *target, nil
}

// requireSuperuser reports whether user is a superuser.
//
// It can return errors annotated with [api.Unauthorized] or [api.Forbidden].
func requireSuperuser(user *api.UserInfo) error {
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

const rootUsername = "root"

// EnsureRootUser ensures that there is a user named "root" in the system.
// If there is no such user, it creates a new root user with superuser privileges, and generates and logs out an API key.
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

	_, err = s.store.CreateUser(ctx, api.UserCreateRequest{
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
		"created root superuser with bootstrap API key; store this key securely and revoke it after first use",
		slog.String("username", rootUsername),
		slog.String("key", issued.Key),
	)
	return nil
}
