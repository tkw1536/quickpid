//spellchecker:words backend
package backend

//spellchecker:words context errors github bicpid
import (
	"context"
	"errors"

	"github.com/tkw1536/bicpid/api"
)

// AuthorizationBackend represents the backend for per-namespace permissions.
//
// See [memory.NewStore] and [gorm.NewStore] for implementations.
type AuthorizationBackend interface {
	// Gets the permission level for a username in a namespace.
	// Returns [api.PermissionLevelNone] if no explicit permission is stored.
	GetNamespacePermission(ctx context.Context, namespace, username string) (api.PermissionLevel, error)

	// Sets or updates the permission level for a username in a namespace.
	// Setting [api.PermissionLevelNone] should remove any explicit permission record.
	SetNamespacePermission(ctx context.Context, namespace, username string, level api.PermissionLevel) error

	// Deletes an explicit permission record.
	// Should return [ErrPermissionNotFound] if no explicit permission exists.
	DeleteNamespacePermission(ctx context.Context, namespace, username string) error

	// Lists users with explicit non-none permissions in a namespace, ordered ascending by username.
	// Has no specific error conditions.
	ListNamespacePermissions(ctx context.Context, namespace string, params api.ListNamespacePermissionsParams) (*api.PaginatedNamespacePermissionsResponse, error)

	Shutdowner
}

// Sentinel errors to be returned by [AuthorizationBackend] implementations.
var (
	ErrPermissionNotFound     = errors.New("permission not found")
	ErrInvalidPermissionLevel = errors.New("invalid permission level")
)
