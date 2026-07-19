//spellchecker:words backend
package backend

//spellchecker:words context errors time github quickpid
import (
	"context"
	"errors"
	"time"

	"github.com/tkw1536/quickpid/api"
)

// ResolverBackend represents storage for the low-level PID resolver.
//
//spellchecker:words context errors time github quickpid
type ResolverBackend interface {
	// Lists all available namespaces that the given user has some access to and that are ordered ascending by namespace id.
	// User may be omitted in which case all namespaces should be considered.
	//
	// Should return [ErrUserNotFound] if owner does not exist.
	ListNamespaces(ctx context.Context, user *api.ValidUsername, params api.ListNamespacesParams) (*api.PaginatedNamespacesResponse, error)

	// Creates a new namespace and grants manager permissions for the owner.
	// If owner is nil, no new manager permissions are granted.
	//
	// Should return [ErrDuplicateNamespaceID] if the namespace id is already in use.
	// Should return [ErrUserNotFound] if owner does not exist.
	CreateNamespace(ctx context.Context, namespace *api.ValidNamespaceID, req api.NamespaceCreateRequest, owner *api.ValidUsername, now func() time.Time) (*api.NamespaceResponse, error)

	// Gets a namespace by its identifier.
	// Should return [ErrNamespaceNotFound] if the namespace is not found.
	GetNamespace(ctx context.Context, namespace *api.ValidNamespaceID) (*api.NamespaceResponse, error)

	// Lists all resources in a namespace, ordered ascending by pid.
	// Should return [ErrNamespaceNotFound] if the namespace is not found.
	ListResources(ctx context.Context, namespace *api.ValidNamespaceID, params api.ListResourcesParams) (*api.PaginatedResourcesResponse, error)

	// Counts the number of resources across all namespaces, including soft-deleted resources.
	// Has no specific error conditions.
	CountAllResources(ctx context.Context) (int64, error)

	// Gets a resource by the given namespace and pid.
	//
	// Should return [ErrNamespaceNotFound] if the namespace is not found.
	// Should return [ErrResourceNotFound] if the resource is not found.
	GetResource(ctx context.Context, namespace *api.ValidNamespaceID, pid *api.ValidPID) (*api.ResourceResponse, error)

	// Creates a new resource in the given namespace with the given pid.
	//
	// Should return [ErrNamespaceNotFound] if the namespace is not found.
	// Should return [ErrPIDAllocationFailed] if the pid is already in use.
	CreateResource(ctx context.Context, namespace *api.ValidNamespaceID, pid *api.ValidPID, req api.ResourceCreateRequest, now func() time.Time) (*api.ResourceResponse, error)

	// Creates multiple resources in the given namespace with the given pids.
	// If the creation of a single resource fails, should roll back the entire batch.
	//
	// Should return [ErrNamespaceNotFound] if the namespace is not found.
	// Should return [ErrPIDAllocationFailed] if one of the pids is already in use.
	BatchCreateResources(ctx context.Context, namespace *api.ValidNamespaceID, pids []*api.ValidPID, reqs []api.ResourceCreateRequest, now func() time.Time) ([]api.ResourceResponse, error)

	// Updates a resource in the given namespace with the given pid.
	//
	// Should return [ErrNamespaceNotFound] if the namespace is not found.
	// Should return [ErrResourceNotFound] if the resource did not previously exist.
	UpdateResource(ctx context.Context, namespace *api.ValidNamespaceID, pid *api.ValidPID, req api.ResourceUpdateRequest, now func() time.Time) (*api.ResourceResponse, error)

	WithShutdownMethod
}

// Sentinel errors to be returned by [ResolverBackend] implementations.
var (
	ErrDuplicateNamespaceID = errors.New("duplicate namespace id")

	ErrNamespaceNotFound = errors.New("namespace not found")
	ErrResourceNotFound  = errors.New("resource not found")

	ErrPIDAllocationFailed = errors.New("could not allocate unique pid")
)
