package api

import "errors"

// Sentinel errors returned by Resolver implementations and expected by the HTTP layer.
// Match with errors.Is; map to 4xx in the server handler.
var (
	ErrEmptyRequestBody       = errors.New("empty request body")
	ErrInvalidJSON            = errors.New("invalid JSON")
	ErrTrailingJSON           = errors.New("trailing JSON")
	ErrNamespaceNotFound      = errors.New("namespace not found")
	ErrResourceNotFound       = errors.New("resource not found")
	ErrNamespaceAlreadyExists = errors.New("namespace already exists")
	// ErrPIDAllocationFailed is returned when a unique PID could not be allocated after retries.
	ErrPIDAllocationFailed = errors.New("could not allocate unique pid")
)
