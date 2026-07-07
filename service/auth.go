//spellchecker:words service
package service

//spellchecker:words context crypto rand errors slog github bicpid backend internal apikey
import (
	"context"
	"crypto/rand"
	"errors"
	"log/slog"

	"github.com/tkw1536/bicpid/api"
	"github.com/tkw1536/bicpid/backend"
	"github.com/tkw1536/bicpid/internal/apikey"
)

// Authenticate looks up the username for a valid API key.
// It returns errUnauthorized when the key is missing or invalid.
func (s *Service) Authenticate(ctx context.Context, apiKey string) (string, error) {
	if apiKey == "" {
		return "", errUnauthorized
	}

	username, err := s.auth.LookupUserByKey(ctx, apiKey)
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
	caller, err := s.Authenticate(ctx, apiKey)
	if err != nil {
		return nil, err
	}
	user, err := s.auth.GetUser(ctx, caller)
	if errors.Is(err, backend.ErrUserNotFound) {
		return nil, errUnauthorized
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUser returns the user account for caller.
func (s *Service) GetUser(ctx context.Context, caller string) (*api.UserInfo, api.Error, error) {
	user, err := s.auth.GetUser(ctx, caller)
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
	if err := requireSuperuser(caller); errors.Is(err, errForbidden) {
		return nil, "", err
	}
	if req.Superuser && !caller.Superuser {
		return nil, "", errForbidden
	}

	user, err := s.auth.CreateUser(ctx, req, s.runtime.Now)
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
	if err := requireSuperuser(caller); errors.Is(err, errForbidden) {
		return nil, "", err
	}

	page, err := s.auth.ListUsers(ctx, params)
	if err != nil {
		return nil, api.DatabaseError, err
	}
	return page, "", nil
}

// ListKeys lists API keys for caller.
func (s *Service) ListKeys(ctx context.Context, caller *api.UserInfo, params api.ListKeysParams) (*api.PaginatedAPIKeysResponse, api.Error, error) {
	page, err := s.auth.ListKeys(ctx, caller.Username, params)
	if specError, ok := mapAuthBackendError(err); ok {
		return nil, specError, err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}
	return page, "", nil
}

// IssueKey issues a new API key for caller.
func (s *Service) IssueKey(ctx context.Context, caller *api.UserInfo, req api.IssueKeyRequest) (*api.IssueKeyResponse, api.Error, error) {
	keyID, err := s.runtime.NewNamespaceID()
	if err != nil {
		return nil, api.BadIDGeneration, err
	}

	issued, err := s.createIssuedKey(ctx, caller.Username, keyID, req)
	if specError, ok := mapAuthBackendError(err); ok {
		return nil, specError, err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}
	return issued, "", nil
}

func (s *Service) createIssuedKey(ctx context.Context, username, keyID string, req api.IssueKeyRequest) (*api.IssueKeyResponse, error) {
	rawKey, err := apikey.Default.Generate(rand.Reader)
	if err != nil {
		return nil, err
	}
	info, err := s.auth.CreateKey(ctx, username, keyID, rawKey, req, s.runtime.Now)
	if err != nil {
		return nil, err
	}
	return &api.IssueKeyResponse{APIKeyInfo: *info, Key: rawKey}, nil
}

// RevokeKey revokes an API key for caller.
func (s *Service) RevokeKey(ctx context.Context, caller *api.UserInfo, req api.RevokeKeyRequest) (*api.RevokeKeyResponse, api.Error, error) {
	info, err := s.auth.RevokeKey(ctx, caller.Username, req.ID)
	if specError, ok := mapAuthBackendError(err); ok {
		return nil, specError, err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}
	return &api.RevokeKeyResponse{APIKeyInfo: *info}, "", nil
}

// UpdateUser updates caller's account or another user when caller is a superuser.
func (s *Service) UpdateUser(ctx context.Context, caller *api.UserInfo, req api.UpdateUserRequest) (*api.UserInfo, api.Error, error) {
	target := caller.Username
	if req.Username != nil {
		target = *req.Username
	}

	// only own account can be updated, or the user needs to be a superuser.
	if target != caller.Username && !caller.Superuser {
		return nil, "", errForbidden
	}

	// cannot change superuser flag, unless the current user is a superuser.
	if req.Superuser != nil && !caller.Superuser {
		return nil, "", errForbidden
	}

	user, err := s.auth.UpdateUser(ctx, target, req)
	if specError, ok := mapAuthBackendError(err); ok {
		return nil, specError, err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}
	return user, "", nil
}

func requireSuperuser(user *api.UserInfo) error {
	if !user.Superuser {
		return errForbidden
	}
	return nil
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

const rootKeyID = "bootstrap"

// EnsureRootUser creates a root superuser with a bootstrap API key when no accounts exist.
func (s *Service) EnsureRootUser(ctx context.Context, logger *slog.Logger) error {
	page, err := s.auth.ListUsers(ctx, api.ListUsersParams{Limit: 1})
	if err != nil {
		return err
	}
	if page.Total > 0 {
		return nil
	}

	_, err = s.auth.CreateUser(ctx, api.UserCreateRequest{
		Username:  rootUsername,
		Superuser: true,
	}, s.runtime.Now)
	if errors.Is(err, backend.ErrDuplicateUsername) {
		return nil
	}
	if err != nil {
		return err
	}

	issued, err := s.createIssuedKey(ctx, rootUsername, rootKeyID, api.IssueKeyRequest{
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
