// Package backend provides [Backend] and implementations.
package backend

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/tkw1536/quickpid/spec"
)

// Backend represents the backend of a PID resolver.
//
// Implementations should return appropriate sentinel errors from below.
type Backend interface {
	ListNamespaces(ctx context.Context, params spec.ListNamespacesParams) (*spec.PaginatedNamespacesResponse, error)
	CreateNamespace(ctx context.Context, namespace string, req spec.NamespaceCreateRequest, now func() time.Time) (*spec.NamespaceResponse, error)

	ListResources(ctx context.Context, params spec.ListResourcesParams) (*spec.PaginatedResourcesResponse, error)
	GetResource(ctx context.Context, namespace, pid string) (*spec.ResourceResponse, error)

	CreateResource(ctx context.Context, namespace string, req spec.ResourceCreateRequest, rand io.Reader, now func() time.Time) (*spec.ResourceResponse, error)
	BatchCreateResources(ctx context.Context, namespace string, reqs []spec.ResourceCreateRequest, rand io.Reader, now func() time.Time) ([]spec.ResourceResponse, error)

	UpdateResource(ctx context.Context, namespace, pid string, req spec.ResourceUpdateRequest, now func() time.Time) (*spec.ResourceResponse, error)
}

// Sentinel errors to be returned by [Backend] implementations.
var (
	ErrEmptyRequestBody = errors.New("empty request body")
	ErrInvalidJSON      = errors.New("invalid JSON")
	ErrTrailingJSON     = errors.New("trailing JSON")

	ErrInvalidQueryParameter = errors.New("invalid query parameter")

	ErrRequestBodyTooLarge = errors.New("request payload too large")
	ErrTooManyItems        = errors.New("too many items")

	ErrNamespaceNotFound = errors.New("namespace not found")
	ErrResourceNotFound  = errors.New("resource not found")

	ErrNamespaceIDAllocationFailed = errors.New("could not allocate unique namespace id")
	ErrInvalidNamespace            = errors.New("invalid namespace")
	ErrInvalidPID                  = errors.New("invalid pid")

	ErrPIDAllocationFailed = errors.New("could not allocate unique pid")
)
