// Package backend provides [Backend] and implementations.
package backend

import (
	"context"
	"errors"
	"time"

	"github.com/tkw1536/quickpid/api"
)

// Backend represents the backend of a PID resolver.
//
// See [NewGormBackend] and [NewInMemoryBackend] for implementations.
type Backend interface {
	// Lists all available namespaces, ordered ascending by namespace id.
	// Has no specific error conditions.
	ListNamespaces(ctx context.Context, params api.ListNamespacesParams) (*api.PaginatedNamespacesResponse, error)

	// Create a new namespace with the given identifier.
	// Should return [ErrDuplicateNamespaceID] if the namespace id is already in use.
	CreateNamespace(ctx context.Context, namespace string, req api.NamespaceCreateRequest, now func() time.Time) (*api.NamespaceResponse, error)

	// Retrieve a namespace by its identifier.
	// Should return [ErrNamespaceNotFound] if the namespace is not found.
	GetNamespace(ctx context.Context, namespace string) (*api.NamespaceResponse, error)

	// Lists all resources in a namespace ordered ascending by pid.
	// Returns [ErrNamespaceNotFound] if the namespace is not found.
	ListResources(ctx context.Context, params api.ListResourcesParams) (*api.PaginatedResourcesResponse, error)

	// CountAllResources returns the number of resources across all namespaces (including soft-deleted).
	CountAllResources(ctx context.Context) (int64, error)

	// Retrieves a resource using the given namespace and pid.
	// Returns [ErrNamespaceNotFound] if the namespace is not found.
	// Returns [ErrResourceNotFound] if the resource is not found.
	GetResource(ctx context.Context, namespace, pid string) (*api.ResourceResponse, error)

	// Creates a new resource in the given namespace with the given pid.
	// Returns [ErrNamespaceNotFound] if the namespace is not found.
	// Returns [ErrPIDAllocationFailed] if the pid is already in use.
	CreateResource(ctx context.Context, namespace, pid string, req api.ResourceCreateRequest, now func() time.Time) (*api.ResourceResponse, error)

	// Creates multiple resources in the given namespace with the given pids.
	// If the creation of a single resource fails, the entire batch is expected to be rolled back.
	//
	// Returns [ErrNamespaceNotFound] if the namespace is not found.
	// Returns [ErrPIDAllocationFailed] if one of the pids is already in use.
	BatchCreateResources(ctx context.Context, namespace string, pids []string, reqs []api.ResourceCreateRequest, now func() time.Time) ([]api.ResourceResponse, error)

	// Updates a resource in the given namespace with the given pid.
	//
	// Returns [ErrNamespaceNotFound] if the namespace is not found.
	// Returns [ErrResourceNotFound] if the resource did not previously exist.
	UpdateResource(ctx context.Context, namespace, pid string, req api.ResourceUpdateRequest, now func() time.Time) (*api.ResourceResponse, error)

	// Close instructs this backend to initiate shutdown.
	// It should wait until the shutdown is complete, or ctx is done, whichever happens first.
	Shutdown(ctx context.Context) error
}

// Sentinel errors to be returned by [Backend] implementations.
var (
	ErrDuplicateNamespaceID = errors.New("duplicate namespace id")

	ErrNamespaceNotFound = errors.New("namespace not found")
	ErrResourceNotFound  = errors.New("resource not found")

	ErrPIDAllocationFailed = errors.New("could not allocate unique pid") // ???
)
