package spec

import "net/http"

// Error represents a specific API error.
type Error string

const (
	InvalidLimitParameter   Error = "invalid_parameter_limit"
	InvalidOffsetParameter  Error = "invalid_parameter_offset"
	InvalidDeletedParameter Error = "invalid_parameter_deleted"

	DatabaseError Error = "database_error" // An internal problem occurred while interacting with the database.

	BodySizeExceeded Error = "body_size_exceeded"
	BodyIsEmpty      Error = "body_missing"
	BodyTrailingJSON Error = "body_trailing_json"
	BodyInvalidJSON  Error = "body_invalid_json"

	NamespaceIDGenerationError     Error = "namespace_id_generation_error"         // An error occurred while generating a namespace ID
	InsufficientNamespaceIDEntropy Error = "insufficient_entropy_for_namespace_id" // Insufficient entropy for namespace ID generation

	PIDGenerationError     Error = "pid_generation_error"         // An error occurred while generating a PID
	InsufficientPIDEntropy Error = "insufficient_entropy_for_pid" // Insufficient entropy for PID generation

	InvalidNamespaceID Error = "invalid_namespace_id"
	InvalidPID         Error = "invalid_pid"
	NamespaceNotFound  Error = "namespace_not_found"
	ResourceNotFound   Error = "resource_not_found"

	TooManyItems Error = "too_many_items"
)

// codes maps [Error]s to HTTP status codes.
var codes = map[Error]int{
	InvalidLimitParameter:   http.StatusBadRequest,
	InvalidOffsetParameter:  http.StatusBadRequest,
	InvalidDeletedParameter: http.StatusBadRequest,

	DatabaseError: http.StatusInternalServerError,

	BodySizeExceeded: http.StatusRequestEntityTooLarge,
	BodyIsEmpty:      http.StatusBadRequest,
	BodyTrailingJSON: http.StatusBadRequest,
	BodyInvalidJSON:  http.StatusBadRequest,

	NamespaceIDGenerationError:     http.StatusInternalServerError,
	InsufficientNamespaceIDEntropy: http.StatusInternalServerError,

	PIDGenerationError:     http.StatusInternalServerError,
	InsufficientPIDEntropy: http.StatusInternalServerError,

	InvalidNamespaceID: http.StatusBadRequest,
	InvalidPID:         http.StatusBadRequest,
	NamespaceNotFound:  http.StatusNotFound,
	ResourceNotFound:   http.StatusNotFound,

	TooManyItems: http.StatusBadRequest,
}

// HTTPCode returns the HTTP status code for the error.
func (e Error) HTTPCode() int {
	code, ok := codes[e]
	if !ok {
		panic("never reached: invalid error code")
	}
	return code
}
