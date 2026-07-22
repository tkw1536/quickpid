package api

//spellchecker:words errors http
import (
	"errors"
	"fmt"
	"net/http"
)

type ErrorResponse struct {
	Error ErrorString `json:"error"`
}

// ErrorString represents an error string from the api.
type ErrorString string

const (
	DatabaseError       ErrorString = "database_error"       // An internal problem with the database
	BadIDGeneration     ErrorString = "bad_id_generation"    // Server failed to generate a valid api key, namespace id or pid
	InsufficientEntropy ErrorString = "insufficient_entropy" // Insufficient entropy for api, namespace or pid generation

	BodyMissing      ErrorString = "body_missing"       // request body was missing (but it was required)
	BodySizeExceeded ErrorString = "body_size_exceeded" // request body size limit exceeded
	BodyInvalidJSON  ErrorString = "body_invalid_json"  // request body did not contain JSON, or it was not in the expected format

	InvalidQueryParameter ErrorString = "invalid_query_parameter" // A query parameter that was sent was invalid

	ItemLimitExceeded ErrorString = "item_limit_exceeded" // The number of items in the request exceeded the limit

	InvalidNamespaceID ErrorString = "invalid_namespace_id" // An invalid namespace id was sent
	InvalidPID         ErrorString = "invalid_pid"          // An invalid pid was sent
	InvalidUsername    ErrorString = "invalid_username"     // An invalid username was sent
	InvalidPassword    ErrorString = "invalid_password"     // An invalid password was sent
	InvalidBaseURI     ErrorString = "invalid_base_uri"     // An invalid base URI was sent

	NamespaceNotFound ErrorString = "namespace_not_found" // Namespace not found
	ResourceNotFound  ErrorString = "resource_not_found"  // Resource not found
	ResourceGone      ErrorString = "resource_gone"       // Resource exists but has been deleted
	MountNotFound     ErrorString = "mount_not_found"     // Mount not found

	PermissionNotFound     ErrorString = "permission_not_found"     // Permission record not found
	InvalidPermissionLevel ErrorString = "invalid_permission_level" // Permission level is not allowed for this operation

	InfoUnavailable ErrorString = "info_unavailable" // Info is unavailable (possibly for security reasons)

	UnavailableInAnonymousMode ErrorString = "unavailable_in_anonymous_mode" // Route is not available in anonymous mode

	DuplicateUsername ErrorString = "duplicate_username" // Username is already in use
	UserNotFound      ErrorString = "user_not_found"     // User not found
	KeyNotFound       ErrorString = "key_not_found"      // API key not found

	Unauthorized ErrorString = "unauthorized" // Authentication is required or the bearer token is invalid

	Forbidden ErrorString = "forbidden" // The authenticated user is not allowed to perform this action
)

// WithErrorString annotates err with the given [ErrorString].
//
// The error message remains unchanged.
// The new error can be Unwrapped to get the underlying error.
func WithErrorString(err error, message ErrorString) error {
	return withStringError{message: message, err: err}
}

// GetErrorString extracts an [ErrorString] that was added to the err using [WithErrorString].
// If no [ErrorString] was added, the second return value is false.
func GetErrorString(err error) (ErrorString, bool) {
	annotated, ok := errors.AsType[withStringError](err)
	return annotated.message, ok
}

type withStringError struct {
	message ErrorString
	err     error
}

func (err withStringError) Error() string {
	// we don't just call err.err.Error() here to avoid err.err == nil or panicking.
	// that's better handled by sprintf.
	return fmt.Sprintf("%s", err.err)
}

func (err withStringError) Unwrap() error {
	return err.err
}

// codes maps [ErrorString]s to HTTP status codes.
var codes = map[ErrorString]int{
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
	InvalidUsername:    http.StatusBadRequest,
	InvalidPassword:    http.StatusBadRequest,
	InvalidBaseURI:     http.StatusBadRequest,

	NamespaceNotFound: http.StatusNotFound,
	ResourceNotFound:  http.StatusNotFound,
	ResourceGone:      http.StatusGone,
	MountNotFound:     http.StatusNotFound,

	PermissionNotFound:     http.StatusNotFound,
	InvalidPermissionLevel: http.StatusBadRequest,

	InfoUnavailable: http.StatusNotFound,

	UnavailableInAnonymousMode: http.StatusNotFound,

	DuplicateUsername: http.StatusConflict,
	UserNotFound:      http.StatusNotFound,
	KeyNotFound:       http.StatusNotFound,

	Unauthorized: http.StatusUnauthorized,
	Forbidden:    http.StatusForbidden,
}

// HTTPCode returns the HTTP status code for the error.
//
// If an error code is invalid, it panics.
func (e ErrorString) HTTPCode() int {
	code, ok := codes[e]
	if !ok {
		panic("invalid error code")
	}
	return code
}

// various internal sentinel errors.
var (
	errMissingRequiredField        = errors.New("missing required field")
	errInvalidPermissionLevel      = errors.New("invalid permission level")
	errFailedToUnmarshalFields     = errors.New("failed to unmarshal fields")
	errInvalidPIDFormat            = errors.New("invalid PID format")
	errInvalidResourceTags         = errors.New("invalid resource tags")
	errTagsMustHaveAtLeastTwo      = errors.New("tags must contain at least two items")
	errTagAndTagsMutuallyExclusive = errors.New("tag and tags are mutually exclusive")
	errMissingTagOrTags            = errors.New("missing required field: tag or tags")
)
