//spellchecker:words service
package service

//spellchecker:words context errors slog github quickpid backend
import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
)

var (
	errForbidden = errors.New("forbidden")
)

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

// CreateUser creates a new user account.
//
// It can return the following errors:
//
// - [api.DuplicateUsername]
// - [api.DatabaseError].
func (s *Service) CreateUser(ctx context.Context, caller api.Caller, req api.ValidUserCreateRequest) (*api.UserInfo, error) {
	user, err := s.store.CreateUser(ctx, req, s.runtime.Now)
	if mapped, ok := mapAuthBackendError(err); ok {
		return nil, mapped
	}
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to create user: %w", err), api.DatabaseError)
	}
	return user, nil
}

// DeleteUser deletes a user account.
//
// It can return the following errors:
//
// - [api.Forbidden]
// - [api.UserNotFound]
// - [api.DatabaseError].
func (s *Service) DeleteUser(ctx context.Context, caller api.Caller, target api.ValidUsername) error {
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
// - [api.DatabaseError].
func (s *Service) ListUsers(ctx context.Context, caller api.Caller, params api.ListUsersParams) (*api.PaginatedUsersResponse, error) {
	page, err := s.store.ListUsers(ctx, params)
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("backend failed to list users: %w", err), api.DatabaseError)
	}
	return page, nil
}

// AutocompleteUsers returns usernames matching a prefix.
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

// UpdateUser updates a user account. Caller must be a superuser and cannot update their own account.
//
// It can return the following errors:
//
// - [api.Forbidden]
// - [api.UserNotFound]
// - [api.DatabaseError].
func (s *Service) UpdateUser(ctx context.Context, caller api.Caller, target api.ValidUsername, req api.UserUpdateRequest) (*api.UserInfo, error) {
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
