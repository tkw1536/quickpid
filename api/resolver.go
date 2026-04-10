package api

import "context"

// Resolver is implemented by backends; one method per OpenAPI operation.
type Resolver interface {
	ListNamespaces(ctx context.Context) ([]NamespaceResponse, error)
	CreateNamespace(ctx context.Context, req NamespaceCreateRequest) (*NamespaceResponse, error)
	ListResources(ctx context.Context, params ListResourcesParams) ([]ResourceResponse, error)
	// CreateResource persists req under namespace using pid (caller-generated, e.g. from server PID policy).
	CreateResource(ctx context.Context, namespace string, req ResourceCreateRequest, pid string) (*ResourceResponse, error)
	// BatchCreateResources persists each request with the matching pids[i]; len(pids) must equal len(reqs).
	BatchCreateResources(ctx context.Context, namespace string, reqs []ResourceCreateRequest, pids []string) ([]ResourceResponse, error)
	GetResource(ctx context.Context, namespace, pid string) (*ResourceResponse, error)
	UpdateResource(ctx context.Context, namespace, pid string, req ResourceUpdateRequest) (*ResourceResponse, error)
}
