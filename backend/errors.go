package backend

import (
	"errors"
)

// Sentinel errors to be returned by [ResolverBackend] implementations.
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
