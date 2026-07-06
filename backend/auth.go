// Package backend provides [ResolverBackend], [AuthBackend], and implementations.
//
//spellchecker:words backend
package backend

//spellchecker:words context errors time github quickpid
import (
	"context"
	"errors"
	"time"

	"github.com/tkw1536/bicpid/api"
)

// AuthBackend represents the backend for user accounts and API keys.
//
// See [authentication.NewInMemoryBackend] and [authentication.NewGormBackend] for implementations.
type AuthBackend interface {
	// CreateUser creates a new user account.
	// Should return [ErrDuplicateUsername] if the username is already in use.
	CreateUser(ctx context.Context, req api.UserCreateRequest, now func() time.Time) (*api.UserInfo, error)

	// GetUser returns a user account.
	// Should return [ErrUserNotFound] if the user does not exist.
	GetUser(ctx context.Context, username string) (*api.UserInfo, error)

	// ListUsers lists all user accounts, ordered ascending by username.
	ListUsers(ctx context.Context, params api.ListUsersParams) (*api.PaginatedUsersResponse, error)

	// DeleteUser removes a user and all associated API keys.
	// Should return [ErrUserNotFound] if the user does not exist.
	DeleteUser(ctx context.Context, username string) error

	// UpdateUser updates fields on an existing user account.
	// Should return [ErrUserNotFound] if the user does not exist.
	UpdateUser(ctx context.Context, username string, req api.UpdateUserRequest) (*api.UserInfo, error)

	// CreateKey stores a new API key for the given user.
	// key is the full raw API key; only a secure hash of it is stored.
	// Should return [ErrUserNotFound] if the user does not exist.
	CreateKey(ctx context.Context, username, keyID, key string, req api.IssueKeyRequest, now func() time.Time) (*api.APIKeyInfo, error)

	// IssueKey generates a new API key and stores it for the given user.
	// The raw key value is returned only in the response.
	// Should return [ErrUserNotFound] if the user does not exist.
	IssueKey(ctx context.Context, username, keyID string, req api.IssueKeyRequest, now func() time.Time) (*api.IssueKeyResponse, error)

	// ListKeys lists API keys for the given user, ordered ascending by id.
	// Should return [ErrUserNotFound] if the user does not exist.
	ListKeys(ctx context.Context, username string, params api.ListKeysParams) (*api.PaginatedAPIKeysResponse, error)

	// GetKey returns a single API key for the given user.
	// Should return [ErrUserNotFound] if the user does not exist.
	// Should return [ErrKeyNotFound] if the key does not exist.
	GetKey(ctx context.Context, username, keyID string) (*api.APIKeyInfo, error)

	// UpdateKey updates metadata for an existing API key.
	// Should return [ErrUserNotFound] if the user does not exist.
	// Should return [ErrKeyNotFound] if the key does not exist.
	UpdateKey(ctx context.Context, username, keyID string, req api.UpdateKeyRequest, now func() time.Time) (*api.APIKeyInfo, error)

	// RevokeKey removes an API key and returns its final metadata.
	// Should return [ErrUserNotFound] if the user does not exist.
	// Should return [ErrKeyNotFound] if the key does not exist.
	RevokeKey(ctx context.Context, username, keyID string) (*api.APIKeyInfo, error)

	// LookupUserByKey returns the username for a valid API key.
	// Should return [ErrInvalidKey] if no matching key exists.
	LookupUserByKey(ctx context.Context, key string) (string, error)

	// Shutdown instructs this backend to initiate shutdown.
	// It should wait until the shutdown is complete, or ctx is done, whichever happens first.
	Shutdown(ctx context.Context) error
}

// Sentinel errors to be returned by [AuthBackend] implementations.
var (
	ErrDuplicateUsername = errors.New("duplicate username")
	ErrUserNotFound      = errors.New("user not found")
	ErrKeyNotFound       = errors.New("key not found")
	ErrInvalidKey        = errors.New("invalid key")
)
