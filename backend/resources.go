//spellchecker:words backend
package backend

//spellchecker:words context errors time github quickpid
import (
	"context"
	"errors"
	"time"

	"github.com/tkw1536/quickpid/api"
)

// ResourceBackend represents storage for resources.
type ResourceBackend interface {
	// Lists all resources in a namespace, ordered ascending by pid.
	// Should return [ErrNamespaceNotFound] if the namespace is not found.
	ListResources(ctx context.Context, namespace api.ValidNamespaceID, params api.ListResourcesParams) (*api.PaginatedResourcesResponse, error)

	// Counts the number of resources across all namespaces, including soft-deleted resources.
	// Has no specific error conditions.
	CountAllResources(ctx context.Context) (int64, error)

	// Gets a resource by the given namespace and pid.
	//
	// Should return [ErrNamespaceNotFound] if the namespace is not found.
	// Should return [ErrResourceNotFound] if the resource is not found.
	GetResource(ctx context.Context, namespace api.ValidNamespaceID, pid api.ValidPID) (*api.ResourceResponse, error)

	// Creates a new resource in the given namespace with the given pid.
	//
	// Should return [ErrNamespaceNotFound] if the namespace is not found.
	// Should return [ErrPIDAllocationFailed] if the pid is already in use.
	CreateResource(ctx context.Context, namespace api.ValidNamespaceID, pid api.ValidPID, req api.ValidResourceCreateRequest, now func() time.Time) (*api.ResourceResponse, error)

	// Creates multiple resources in the given namespace with the given pids.
	// If the creation of a single resource fails, should roll back the entire batch.
	//
	// Should return [ErrNamespaceNotFound] if the namespace is not found.
	// Should return [ErrPIDAllocationFailed] if one of the pids is already in use.
	BatchCreateResources(ctx context.Context, namespace api.ValidNamespaceID, pids []api.ValidPID, reqs []api.ValidResourceCreateRequest, now func() time.Time) ([]api.ResourceResponse, error)

	// Updates a resource in the given namespace with the given pid.
	//
	// Should return [ErrNamespaceNotFound] if the namespace is not found.
	// Should return [ErrResourceNotFound] if the resource did not previously exist.
	UpdateResource(ctx context.Context, namespace api.ValidNamespaceID, pid api.ValidPID, req api.ValidResourceUpdateRequest, now func() time.Time) (*api.ResourceResponse, error)
}

// Sentinel errors to be returned by [ResourceBackend] implementations.
var (
	ErrResourceNotFound    = errors.New("resource not found")
	ErrPIDAllocationFailed = errors.New("could not allocate unique pid")
)
