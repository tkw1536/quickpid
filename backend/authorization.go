//spellchecker:words backend
package backend

//spellchecker:words context errors github quickpid
import (
	"context"
	"errors"

	"github.com/tkw1536/quickpid/api"
)

// AuthorizationBackend represents the backend for per-namespace permissions.
//
// See [memory.NewStore] and [gorm.NewStore] for implementations.
type AuthorizationBackend interface {
	// Gets the permission level for a username in a namespace.
	//
	// Should return [api.PermissionLevelNone] if no explicit permission is stored.
	GetNamespacePermission(ctx context.Context, namespace *api.ValidNamespaceID, username *api.ValidUsername) (api.PermissionLevel, error)

	// Sets or updates the permission level for a username in a namespace.
	// Setting [api.PermissionLevelNone] should remove any explicit permission record.
	//
	// Should return [ErrInvalidPermissionLevel] if the permission level is invalid.
	// Should return [ErrUserNotFound] if the user does not exist.
	SetNamespacePermission(ctx context.Context, namespace *api.ValidNamespaceID, username *api.ValidUsername, level api.PermissionLevel) error

	// Deletes an explicit permission record.
	//
	// Should return [ErrPermissionNotFound] if no explicit permission exists.
	DeleteNamespacePermission(ctx context.Context, namespace *api.ValidNamespaceID, username *api.ValidUsername) error

	// Lists users with explicit non-none permissions in a namespace, ordered ascending by username.
	//
	// Has no specific error conditions.
	ListNamespacePermissions(ctx context.Context, namespace *api.ValidNamespaceID, params api.ListNamespacePermissionsParams) (*api.PaginatedNamespacePermissionsResponse, error)

	WithShutdownMethod
}

// Sentinel errors to be returned by [AuthorizationBackend] implementations.
var (
	ErrPermissionNotFound     = errors.New("permission not found")
	ErrInvalidPermissionLevel = errors.New("invalid permission level")
)
