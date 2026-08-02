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
	DatabaseError       ErrorString = "databaseError"       // An internal problem with the database
	BadIDGeneration     ErrorString = "badIdGeneration"     // Server failed to generate a valid api key, namespace id or pid
	InsufficientEntropy ErrorString = "insufficientEntropy" // Insufficient entropy for api, namespace or pid generation

	BodyMissing      ErrorString = "bodyMissing"      // request body was missing (but it was required)
	BodySizeExceeded ErrorString = "bodySizeExceeded" // request body size limit exceeded
	BodyInvalidJSON  ErrorString = "bodyInvalidJson"  // request body did not contain JSON, or it was not in the expected format

	InvalidQueryParameter ErrorString = "invalidQueryParameter" // A query parameter that was sent was invalid

	ItemLimitExceeded ErrorString = "itemLimitExceeded" // The number of items in the request exceeded the limit

	InvalidNamespaceID       ErrorString = "invalidNamespaceId"       // An invalid namespace id was sent
	InvalidPID               ErrorString = "invalidPid"               // An invalid pid was sent
	InvalidUsername          ErrorString = "invalidUsername"          // An invalid username was sent
	InvalidAutocompleteQuery ErrorString = "invalidAutocompleteQuery" // An invalid query was sent
	InvalidPassword          ErrorString = "invalidPassword"          // An invalid password was sent
	InvalidBaseURI           ErrorString = "invalidBaseUri"           // An invalid base URI was sent

	NamespaceNotFound ErrorString = "namespaceNotFound" // Namespace not found
	ResourceNotFound  ErrorString = "resourceNotFound"  // Resource not found
	MountNotFound     ErrorString = "mountNotFound"     // Mount not found

	RoleNotFound ErrorString = "roleNotFound" // Role record not found
	InvalidRole  ErrorString = "invalidRole"  // Role is not allowed for this operation

	InfoUnavailable ErrorString = "infoUnavailable" // Info is unavailable (possibly for security reasons)

	UnavailableInAnonymousMode ErrorString = "unavailableInAnonymousMode" // Route is not available in anonymous mode

	DuplicateUsername ErrorString = "duplicateUsername" // Username is already in use
	UserNotFound      ErrorString = "userNotFound"      // User not found
	KeyNotFound       ErrorString = "keyNotFound"       // API key not found

	Unauthorized ErrorString = "unauthorized" // Authentication is required but no valid credentials were provided
	Forbidden    ErrorString = "forbidden"    // The authenticated user is not allowed to perform this action
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

	InvalidNamespaceID:       http.StatusBadRequest,
	InvalidPID:               http.StatusBadRequest,
	InvalidUsername:          http.StatusBadRequest,
	InvalidAutocompleteQuery: http.StatusBadRequest,
	InvalidPassword:          http.StatusBadRequest,
	InvalidBaseURI:           http.StatusBadRequest,

	NamespaceNotFound: http.StatusNotFound,
	ResourceNotFound:  http.StatusNotFound,
	MountNotFound:     http.StatusNotFound,

	RoleNotFound: http.StatusNotFound,
	InvalidRole:  http.StatusBadRequest,

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
	errFailedToUnmarshalFields = errors.New("failed to unmarshal fields")
)

// missingRequiredFieldError is an error that is returned when a required field is missing.
type missingRequiredFieldError string

func (err missingRequiredFieldError) Error() string {
	return fmt.Sprintf("missing required field: %q", string(err))
}
