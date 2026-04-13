package api

import "context"

// Resolver implements the PID Resolver API.
// Implementations should return appropriate sentinel errors.
type Resolver interface {
	ListNamespaces(ctx context.Context, params ListNamespacesParams) (*PaginatedNamespacesResponse, error)
	CreateNamespace(ctx context.Context, req NamespaceCreateRequest) (*NamespaceResponse, error)

	ListResources(ctx context.Context, params ListResourcesParams) (*PaginatedResourcesResponse, error)
	GetResource(ctx context.Context, namespace, pid string) (*ResourceResponse, error)

	CreateResource(ctx context.Context, namespace string, req ResourceCreateRequest, pidGen func() (string, error)) (*ResourceResponse, error)
	BatchCreateResources(ctx context.Context, namespace string, reqs []ResourceCreateRequest, pidGen func() (string, error)) ([]ResourceResponse, error)

	UpdateResource(ctx context.Context, namespace, pid string, req ResourceUpdateRequest) (*ResourceResponse, error)
}
