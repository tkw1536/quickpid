//spellchecker:words backend
package backend

//spellchecker:words context errors time github quickpid
import (
	"context"
	"errors"
	"time"

	"github.com/tkw1536/quickpid/api"
)

// UserBackend represents the backend for user accounts.
type UserBackend interface {
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
}

// Sentinel errors to be returned by [UserBackend] implementations.
var (
	ErrDuplicateUsername = errors.New("duplicate username")
	ErrUserNotFound      = errors.New("user not found")
)
