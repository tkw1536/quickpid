//spellchecker:words backend
package backend

//spellchecker:words context errors time github quickpid internal apikey
import (
	"context"
	"errors"
	"time"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/internal/apikey"
)

// AuthenticationBackend represents the backend for user accounts and API keys.
//
// See [authentication.NewInMemoryBackend] and [authentication.NewGormBackend] for implementations.
type AuthenticationBackend interface {
	// CreateUser creates a new user account.
	//
	// Should return [ErrDuplicateUsername] if the username is already in use.
	CreateUser(ctx context.Context, req api.ValidUserCreateRequest, now func() time.Time) (*api.UserInfo, error)

	// Gets a user account.
	//
	// Should return [ErrUserNotFound] if the user does not exist.
	GetUser(ctx context.Context, username api.ValidUsername) (*api.UserInfo, error)

	// ListUsers lists all user accounts, ordered ascending by username.
	//
	// Has no specific error conditions.
	ListUsers(ctx context.Context, params api.ListUsersParams) (*api.PaginatedUsersResponse, error)

	// AutocompleteUsers returns usernames with the given prefix, ordered ascending by username.
	//
	// Has no specific error conditions.
	AutocompleteUsers(ctx context.Context, query api.ValidAutocompleteQuery, limit int) ([]string, error)

	// DeleteUser removes a user and all associated API keys.
	//
	// Should return [ErrUserNotFound] if the user does not exist.
	DeleteUser(ctx context.Context, username api.ValidUsername) error

	// UpdateUser updates fields on an existing user account.
	//
	// Should return [ErrUserNotFound] if the user does not exist.
	UpdateUser(ctx context.Context, username api.ValidUsername, req api.UserUpdateRequest) (*api.UserInfo, error)

	// SetPassword sets or clears a password for the given user.
	//
	// A nil password clears the stored password.
	// Should return [ErrUserNotFound] if the user does not exist.
	SetPassword(ctx context.Context, username api.ValidUsername, password *api.ValidPassword) (bool, error)

	// CheckPassword reports whether password matches the stored password for the given user.
	//
	// Should return [ErrUserNotFound] if the user does not exist.
	CheckPassword(ctx context.Context, username api.ValidUsername, password api.ValidPassword) (bool, error)

	// Creates a new API key for the given user.
	// format describes how key should be validated and transformed into its stored representation.
	// key is the full raw API key; only a secure hash of it should be stored by the backend.
	//
	// Should return [ErrUserNotFound] if the user does not exist.
	// Should return [ErrKeyCollision] if the key could not be created due to a collision with an existing key.
	CreateKey(ctx context.Context, format apikey.Format, username api.ValidUsername, keyID string, key string, req api.KeyIssueRequest, now func() time.Time) (*api.APIKeyInfo, error)

	// ListKeys lists API keys for the given user, ordered ascending by id.
	// The key list may include keys that are no longer valid.
	//
	// Should return [ErrUserNotFound] if the user does not exist.
	ListKeys(ctx context.Context, format apikey.Format, username api.ValidUsername, params api.ListKeysParams) (*api.PaginatedAPIKeysResponse, error)

	// Revokes an API key.
	// Revoking an already-revoked key succeeds (idempotent).
	//
	// Should return [ErrUserNotFound] if the user does not exist.
	// Should return [ErrKeyNotFound] if the key does not exist.
	RevokeKey(ctx context.Context, format apikey.Format, username api.ValidUsername, keyID string) error

	// Looks up an api and returns it along with it's username.
	// The returned key may or may not be valid.
	//
	// Should return [ErrInvalidKey] if no matching key exists.
	LookupUserByKey(ctx context.Context, format apikey.Format, key string) (string, *api.APIKeyInfo, error)

	WithShutdownMethod
}

// Sentinel errors to be returned by [AuthenticationBackend] implementations.
var (
	ErrDuplicateUsername = errors.New("duplicate username")
	ErrUserNotFound      = errors.New("user not found")
	ErrKeyNotFound       = errors.New("key not found")
	ErrInvalidKey        = errors.New("invalid key")
	ErrKeyCollision      = errors.New("api key collides with existing key")
)
