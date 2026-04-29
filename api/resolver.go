// Package api provides go implementations for the PID Resolver API.
package api

import (
	"context"
	"io"
	"time"
)

// ResolverBackend represents a resolver backend from go.
type ResolverBackend interface {
	ListNamespaces(ctx context.Context, params ListNamespacesParams) (*PaginatedNamespacesResponse, error)
	CreateNamespace(ctx context.Context, namespace string, req NamespaceCreateRequest, now func() time.Time) (*NamespaceResponse, error)

	ListResources(ctx context.Context, params ListResourcesParams) (*PaginatedResourcesResponse, error)
	GetResource(ctx context.Context, namespace, pid string) (*ResourceResponse, error)

	CreateResource(ctx context.Context, namespace string, req ResourceCreateRequest, rand io.Reader, now func() time.Time) (*ResourceResponse, error)
	BatchCreateResources(ctx context.Context, namespace string, reqs []ResourceCreateRequest, rand io.Reader, now func() time.Time) ([]ResourceResponse, error)

	UpdateResource(ctx context.Context, namespace, pid string, req ResourceUpdateRequest, now func() time.Time) (*ResourceResponse, error)
}
