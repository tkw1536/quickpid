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

// CredentialsBackend represents the backend for passwords and API keys.
type CredentialsBackend interface {
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

	// Revokes an API key; revoking a key should delete it from the backend entirely.
	// Background jobs typically revoke expired API keys in order to trigger deletion.
	//
	// Should return [ErrUserNotFound] if the user does not exist.
	// Should return [ErrKeyNotFound] if the key does not exist.
	RevokeKey(ctx context.Context, format apikey.Format, username api.ValidUsername, keyID string) error

	// Looks up an API key and returns it along with it's username.
	// The returned key may or may not be valid.
	//
	// Should return [ErrInvalidKey] if no matching key exists.
	LookupUserByKey(ctx context.Context, format apikey.Format, key string) (string, *api.APIKeyInfo, error)
}

// Sentinel errors to be returned by [CredentialsBackend] implementations.
var (
	ErrKeyNotFound  = errors.New("key not found")
	ErrInvalidKey   = errors.New("invalid key")
	ErrKeyCollision = errors.New("api key collides with existing key")
)
