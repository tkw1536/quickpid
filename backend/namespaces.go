//spellchecker:words backend
package backend

//spellchecker:words context errors time github quickpid
import (
	"context"
	"errors"
	"time"

	"github.com/tkw1536/quickpid/api"
)

// NamespaceBackend represents storage for namespaces.
type NamespaceBackend interface {
	// Lists all available namespaces ordered ascending by namespace id.
	ListNamespaces(ctx context.Context, params api.ListNamespacesParams) (*api.PaginatedNamespacesResponse, error)

	// Creates a new namespace and grants the manager role to the owner.
	// If owner is nil, no manager role is granted.
	//
	// Should return [ErrDuplicateNamespaceID] if the namespace id is already in use.
	// Should return [ErrUserNotFound] if owner does not exist.
	CreateNamespace(ctx context.Context, namespace api.ValidNamespaceID, req api.NamespaceCreateRequest, owner *api.ValidUsername, now func() time.Time) (*api.NamespaceResponse, error)

	// Gets a namespace by its identifier.
	// Should return [ErrNamespaceNotFound] if the namespace is not found.
	GetNamespace(ctx context.Context, namespace api.ValidNamespaceID) (*api.NamespaceResponse, error)

	// Updates a namespace by its identifier.
	// Should return [ErrNamespaceNotFound] if the namespace is not found.
	UpdateNamespace(ctx context.Context, namespace api.ValidNamespaceID, req api.ValidNamespaceUpdateRequest, now func() time.Time) (*api.NamespaceResponse, error)
}

// Sentinel errors to be returned by [NamespaceBackend] implementations.
var (
	ErrDuplicateNamespaceID = errors.New("duplicate namespace id")
	ErrNamespaceNotFound    = errors.New("namespace not found")
)
