// Package backend provides [ResolverBackend], [AuthBackend], [AuthorizationBackend], and implementations.
//
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
	// GetNamespacePermission returns the permission level for username in namespace.
	// Returns [api.PermissionLevelNone] when no explicit permission is stored.
	GetNamespacePermission(ctx context.Context, namespace, username string) (api.PermissionLevel, error)

	// SetNamespacePermission sets or updates the permission level for username in namespace.
	// Setting [api.PermissionLevelNone] removes any explicit permission record.
	SetNamespacePermission(ctx context.Context, namespace, username string, level api.PermissionLevel) error

	// DeleteNamespacePermission removes an explicit permission record.
	// Returns [ErrPermissionNotFound] when no explicit permission exists.
	DeleteNamespacePermission(ctx context.Context, namespace, username string) error

	// ListNamespacePermissions lists users with explicit non-none permissions in a namespace,
	// ordered ascending by username.
	ListNamespacePermissions(ctx context.Context, namespace string, params api.ListNamespacePermissionsParams) (*api.PaginatedNamespacePermissionsResponse, error)

	// Shutdown instructs this backend to initiate shutdown.
	Shutdown(ctx context.Context) error
}

// Sentinel errors to be returned by [AuthorizationBackend] implementations.
var (
	ErrPermissionNotFound    = errors.New("permission not found")
	ErrInvalidPermissionLevel = errors.New("invalid permission level")
)
