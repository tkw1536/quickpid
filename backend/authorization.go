//spellchecker:words backend
package backend

//spellchecker:words context errors github quickpid
import (
	"context"
	"errors"

	"github.com/tkw1536/quickpid/api"
)

// AuthorizationBackend represents the backend for per-namespace roles.
//
// See [memory.NewStore] and [gorm.NewStore] for implementations.
type AuthorizationBackend interface {
	// Gets the role for a username in a namespace.
	//
	// Should return [api.RoleNone] if no explicit role is stored.
	GetNamespaceRole(ctx context.Context, namespace api.ValidNamespaceID, username api.ValidUsername) (api.Role, error)

	// Sets or updates the role for a username in a namespace.
	// Setting [api.RoleNone] should remove any explicit role record.
	//
	// Should return [ErrInvalidRole] if the role is invalid.
	// Should return [ErrUserNotFound] if the user does not exist.
	SetNamespaceRole(ctx context.Context, namespace api.ValidNamespaceID, username api.ValidUsername, role api.Role) error

	// Deletes an explicit role record.
	//
	// Should return [ErrRoleNotFound] if no explicit role exists.
	DeleteNamespaceRole(ctx context.Context, namespace api.ValidNamespaceID, username api.ValidUsername) error

	// Lists users with explicit non-none roles in a namespace, ordered ascending by username.
	//
	// Has no specific error conditions.
	ListNamespaceRoles(ctx context.Context, namespace api.ValidNamespaceID, params api.ListNamespaceRolesParams) (*api.PaginatedNamespaceRolesResponse, error)

	// Lists namespaces where the user has an explicit non-none role, ordered ascending by namespace.
	//
	// Should return [ErrUserNotFound] if the user does not exist.
	ListUserRoles(ctx context.Context, username api.ValidUsername, params api.ListUserRolesParams) (*api.PaginatedUserRolesResponse, error)

	WithShutdownMethod
}

// Sentinel errors to be returned by [AuthorizationBackend] implementations.
var (
	ErrRoleNotFound = errors.New("role not found")
	ErrInvalidRole  = errors.New("invalid role")
)
