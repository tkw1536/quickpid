//spellchecker:words service
package service

//spellchecker:words context crypto rand errors slog time github bicpid backend internal apikey
import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/tkw1536/bicpid/api"
	"github.com/tkw1536/bicpid/backend"
	"github.com/tkw1536/bicpid/internal/apikey"
)

// Authenticate looks up the username for a valid API key.
// It returns errUnauthorized when the key is missing or invalid.
func (s *Service) Authenticate(ctx context.Context, apiKey string) (string, error) {
	if s.Options().Anonymous {
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
		return "", err
	}
	return username, nil
}

// CurrentUser authenticates an API key and returns the corresponding user account.
// It returns errUnauthorized when the key is missing, invalid, or the user no longer exists.
func (s *Service) CurrentUser(ctx context.Context, apiKey string) (*api.UserInfo, error) {
	if s.Options().Anonymous {
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
		return nil, err
	}
	return user, nil
}

// GetUserAccount returns the caller's account or another user's when caller is a superuser.
func (s *Service) GetUserAccount(ctx context.Context, caller *api.UserInfo, target *string) (*api.UserInfo, api.Error, error) {
	if specError, err := s.requireAuthEnabled(); err != nil {
		return nil, specError, err
	}
	username, err := resolveTargetUsername(caller, target)
	if err != nil {
		return nil, "", err
	}
	user, err := s.store.GetUser(ctx, username)
	if specError, ok := mapAuthBackendError(err); ok {
		return nil, specError, err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}
	return user, "", nil
}

// GetUser returns the user account for caller.
func (s *Service) GetUser(ctx context.Context, caller string) (*api.UserInfo, api.Error, error) {
	if specError, err := s.requireAuthEnabled(); err != nil {
		return nil, specError, err
	}
	user, err := s.store.GetUser(ctx, caller)
	if specError, ok := mapAuthBackendError(err); ok {
		return nil, specError, err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}
	return user, "", nil
}

// CreateUser creates a new user account. Caller must be a superuser.
func (s *Service) CreateUser(ctx context.Context, caller *api.UserInfo, req api.UserCreateRequest) (*api.UserInfo, api.Error, error) {
	if specError, err := s.requireAuthEnabled(); err != nil {
		return nil, specError, err
	}
	if err := requireSuperuser(caller); err != nil {
		return nil, "", err
	}
	if err := ValidateUsername(req.Username); err != nil {
		return nil, api.InvalidUsername, err
	}
	if req.Superuser && !caller.Superuser {
		return nil, "", errForbidden
	}

	user, err := s.store.CreateUser(ctx, req, s.runtime.Now)
	if specError, ok := mapAuthBackendError(err); ok {
		return nil, specError, err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}
	return user, "", nil
}

// ListUsers lists all user accounts. Caller must be a superuser.
func (s *Service) ListUsers(ctx context.Context, caller *api.UserInfo, params api.ListUsersParams) (*api.PaginatedUsersResponse, api.Error, error) {
	if specError, err := s.requireAuthEnabled(); err != nil {
		return nil, specError, err
	}
	if err := requireSuperuser(caller); err != nil {
		return nil, "", err
	}

	page, err := s.store.ListUsers(ctx, params)
	if err != nil {
		return nil, api.DatabaseError, err
	}
	return page, "", nil
}

// ListKeys lists API keys for the caller or another user when caller is a superuser.
func (s *Service) ListKeys(ctx context.Context, caller *api.UserInfo, target *string, params api.ListKeysParams) (*api.PaginatedAPIKeysResponse, api.Error, error) {
	if specError, err := s.requireAuthEnabled(); err != nil {
		return nil, specError, err
	}
	username, err := resolveTargetUsername(caller, target)
	if err != nil {
		return nil, "", err
	}
	page, err := s.store.ListKeys(ctx, s.apiKeyFormat(), username, params)
	if specError, ok := mapAuthBackendError(err); ok {
		return nil, specError, err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}
	return page, "", nil
}

// IssueKey issues a new API key for caller or another user when caller is a superuser.
//
// It can return the following errors:
//
// - [api.Forbidden]
// - [api.BadIDGeneration]
// - [api.DatabaseError]
// - [api.InsufficientEntropy]
func (s *Service) IssueKey(ctx context.Context, caller *api.UserInfo, target *string, req api.KeyIssueRequest) (*api.IssueKeyResponse, api.Error, error) {
	if specError, err := s.requireAuthEnabled(); err != nil {
		return nil, specError, err
	}
	username, err := resolveTargetUsername(caller, target)
	if err != nil {
		return nil, "", err
	}
	if req.ExpiresAt != nil {
		expiresAt, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			return nil, api.InvalidQueryParameter, err
		}
		if !expiresAt.After(s.runtime.Now()) {
			return nil, api.InvalidQueryParameter, errExpiresAtInPast
		}
	}

	if _, err := s.store.GetUser(ctx, username); err != nil {
		if specError, ok := mapAuthBackendError(err); ok {
			return nil, specError, err
		}
		return nil, api.DatabaseError, err
	}

	issued, specError, err := s.issueAPIKey(ctx, username, req)
	if err != nil {
		return nil, specError, err
	}
	return issued, "", nil
}

// issueAPIKey issues a new API key, retrying when storage reports [backend.ErrKeyCollision].
//
// It can return the following errors:
//
// - [api.BadIDGeneration]
// - [api.DatabaseError]
// - [api.InsufficientEntropy]
func (s *Service) issueAPIKey(ctx context.Context, username string, req api.KeyIssueRequest) (*api.IssueKeyResponse, api.Error, error) {
	s.mu.RLock()
	maxAttempts := s.opts.Limits.MaxAPIKeyAttempts
	s.mu.RUnlock()

	format := s.apiKeyFormat()
	for range maxAttempts {
		keyID, err := s.runtime.NewAPIKeyID()
		if err != nil {
			return nil, api.BadIDGeneration, err
		}

		rawKey, err := s.runtime.NewAPIKey(format)
		if err != nil {
			return nil, api.BadIDGeneration, err
		}

		info, err := s.store.CreateKey(ctx, format, username, keyID, rawKey, req, s.runtime.Now)
		if err == nil {
			return &api.IssueKeyResponse{APIKeyInfo: *info, Key: rawKey}, "", nil
		}
		if specError, ok := mapAuthBackendError(err); ok {
			return nil, specError, err
		}
		if !errors.Is(err, backend.ErrKeyCollision) {
			return nil, api.DatabaseError, err
		}
	}
	return nil, api.InsufficientEntropy, fmt.Errorf("%w: gave up api key generation after %d attempts", errInsufficientEntropy, maxAttempts)
}

// RevokeKey revokes an API key for the caller or another user when caller is a superuser.
func (s *Service) RevokeKey(ctx context.Context, caller *api.UserInfo, target *string, req api.KeyRevokeRequest) (*api.RevokeKeyResponse, api.Error, error) {
	if specError, err := s.requireAuthEnabled(); err != nil {
		return nil, specError, err
	}
	username, err := resolveTargetUsername(caller, target)
	if err != nil {
		return nil, "", err
	}
	info, err := s.store.RevokeKey(ctx, s.apiKeyFormat(), username, req.ID)
	if specError, ok := mapAuthBackendError(err); ok {
		return nil, specError, err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}
	return &api.RevokeKeyResponse{APIKeyInfo: *info}, "", nil
}

// UpdateUser updates a user account. Caller must be a superuser and cannot update their own account.
func (s *Service) UpdateUser(ctx context.Context, caller *api.UserInfo, target string, req api.UserUpdateRequest) (*api.UserInfo, api.Error, error) {
	if specError, err := s.requireAuthEnabled(); err != nil {
		return nil, specError, err
	}
	if err := requireSuperuser(caller); err != nil {
		return nil, "", err
	}
	if target == caller.Username {
		return nil, "", errForbidden
	}

	user, err := s.store.UpdateUser(ctx, target, req)
	if specError, ok := mapAuthBackendError(err); ok {
		return nil, specError, err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}
	return user, "", nil
}

func resolveTargetUsername(caller *api.UserInfo, target *string) (string, error) {
	if target == nil {
		return caller.Username, nil
	}
	if *target != caller.Username && !caller.Superuser {
		return "", errForbidden
	}
	return *target, nil
}

func requireSuperuser(user *api.UserInfo) error {
	if user == nil {
		return errUnauthorized
	}
	if !user.Superuser {
		return errForbidden
	}
	return nil
}

func (s *Service) apiKeyFormat() apikey.Format {
	return apikey.Default
}

func mapAuthBackendError(err error) (api.Error, bool) {
	switch {
	case errors.Is(err, backend.ErrDuplicateUsername):
		return api.DuplicateUsername, true
	case errors.Is(err, backend.ErrUserNotFound):
		return api.UserNotFound, true
	case errors.Is(err, backend.ErrKeyNotFound):
		return api.KeyNotFound, true
	default:
		return "", false
	}
}

const rootUsername = "root"

// EnsureRootUser ensures that there is a user named "root" in the system.
// If there is no such user, it creates a new root user with superuser privileges, and generates and logs out an API key.
func (s *Service) EnsureRootUser(ctx context.Context, logger *slog.Logger) error {
	if s.Options().Anonymous {
		return nil
	}
	page, err := s.store.ListUsers(ctx, api.ListUsersParams{Limit: 1})
	if err != nil {
		return err
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

	issued, _, err := s.issueAPIKey(ctx, rootUsername, api.KeyIssueRequest{
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
