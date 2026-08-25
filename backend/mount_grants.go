//spellchecker:words backend
package backend

//spellchecker:words context errors github quickpid
import (
	"context"
	"errors"

	"github.com/tkw1536/quickpid/api"
)

// MountGrantBackend represents storage for per-user mount-management grants.
//
// A grant is a Go regexp pattern string. A user may manage mounts whose base URI
// matches at least one of their grant patterns (unless they are a superuser).
type MountGrantBackend interface {
	// Lists mount grant patterns for a user, ordered ascending by pattern.
	//
	// Should return [ErrUserNotFound] if the user does not exist.
	ListUserMountGrants(ctx context.Context, username api.ValidUsername, params api.ListUserMountGrantsParams) (*api.PaginatedMountGrantsResponse, error)

	// Creates or replaces a mount grant for a user.
	//
	// Should return [ErrUserNotFound] if the user does not exist.
	SetMountGrant(ctx context.Context, username api.ValidUsername, req api.ValidMountGrantRequest) error

	// Deletes a mount grant for a user.
	//
	// Should return [ErrMountGrantNotFound] if no grant with that pattern exists.
	// Should return [ErrUserNotFound] if the user does not exist.
	DeleteMountGrant(ctx context.Context, username api.ValidUsername, req api.ValidMountGrantRequest) error
}

// Sentinel errors to be returned by [MountGrantBackend] implementations.
var (
	ErrMountGrantNotFound = errors.New("mount grant not found")
)
