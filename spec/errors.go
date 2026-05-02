package spec

import "net/http"

// Error represents a specific API error.
type Error string

const (
	DatabaseError       Error = "database_error"       // An internal problem with the database
	BadIDGeneration     Error = "bad_id_generation"    // Server failed to generate a valid namespace id or pid
	InsufficientEntropy Error = "insufficient_entropy" // Insufficient entropy for namespace or pid generation

	BodyMissing      Error = "body_missing"       // request body was missing (but it was required)
	BodySizeExceeded Error = "body_size_exceeded" // request body size limit exceeded
	BodyInvalidJSON  Error = "body_invalid_json"  // request body did not contain JSON, or it was not in the expected format

	InvalidQueryParameter Error = "invalid_query_parameter" // A query parameter that was sent was invalid

	ItemLimitExceeded Error = "item_limit_exceeded" // The number of items in the request exceeded the limit

	InvalidNamespaceID Error = "invalid_namespace_id" // An invalid namespace id was sent
	InvalidPID         Error = "invalid_pid"          // An invalid pid was sent

	NamespaceNotFound Error = "namespace_not_found" // Namespace not found
	ResourceNotFound  Error = "resource_not_found"  // Resource not found

	InfoUnavailable Error = "info_unavailable" // Info is unavailable (possibly for security reasons)
)

// codes maps [Error]s to HTTP status codes.
var codes = map[Error]int{
	DatabaseError:       http.StatusInternalServerError,
	BadIDGeneration:     http.StatusInternalServerError,
	InsufficientEntropy: http.StatusServiceUnavailable,

	InvalidQueryParameter: http.StatusBadRequest,

	BodyMissing:      http.StatusBadRequest,
	BodySizeExceeded: http.StatusRequestEntityTooLarge,
	BodyInvalidJSON:  http.StatusBadRequest,

	ItemLimitExceeded: http.StatusUnprocessableEntity,

	InvalidNamespaceID: http.StatusBadRequest,
	InvalidPID:         http.StatusBadRequest,

	NamespaceNotFound: http.StatusNotFound,
	ResourceNotFound:  http.StatusNotFound,

	InfoUnavailable: http.StatusNotFound,
}

// HTTPCode returns the HTTP status code for the error.
//
// If an error code is invalid, it panics.
func (e Error) HTTPCode() int {
	code, ok := codes[e]
	if !ok {
		panic("invalid error code")
	}
	return code
}
