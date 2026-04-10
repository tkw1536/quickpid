package api

import "context"

// Resolver is implemented by backends; one method per OpenAPI operation.
type Resolver interface {
	ListNamespaces(ctx context.Context) ([]NamespaceResponse, error)
	CreateNamespace(ctx context.Context, req NamespaceCreateRequest) (*NamespaceResponse, error)
	ListResources(ctx context.Context, params ListResourcesParams) ([]ResourceResponse, error)
	CreateResource(ctx context.Context, namespace string, req ResourceCreateRequest) (*ResourceResponse, error)
	BatchCreateResources(ctx context.Context, namespace string, reqs []ResourceCreateRequest) ([]ResourceResponse, error)
	GetResource(ctx context.Context, namespace, pid string) (*ResourceResponse, error)
	UpdateResource(ctx context.Context, namespace, pid string, req ResourceUpdateRequest) (*ResourceResponse, error)
}
