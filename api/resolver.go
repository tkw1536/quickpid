package api

import "context"

// Resolver is implemented by backends; one method per OpenAPI operation.
type Resolver interface {
	ListNamespaces(ctx context.Context) ([]NamespaceResponse, error)
	CreateNamespace(ctx context.Context, req NamespaceCreateRequest) (*NamespaceResponse, error)
	ListResources(ctx context.Context, params ListResourcesParams) ([]ResourceResponse, error)
	// CreateResource allocates a PID via pidGen (typically from the HTTP handler) and persists req.
	CreateResource(ctx context.Context, namespace string, req ResourceCreateRequest, pidGen func() (string, error)) (*ResourceResponse, error)
	// BatchCreateResources allocates PIDs via pidGen for each item (typically the same function from the handler).
	BatchCreateResources(ctx context.Context, namespace string, reqs []ResourceCreateRequest, pidGen func() (string, error)) ([]ResourceResponse, error)
	GetResource(ctx context.Context, namespace, pid string) (*ResourceResponse, error)
	UpdateResource(ctx context.Context, namespace, pid string, req ResourceUpdateRequest) (*ResourceResponse, error)
}
