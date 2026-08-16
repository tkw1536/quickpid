package api

//spellchecker:words errors http
import (
	"errors"
	"fmt"
	"net/http"
)

// ErrorCode represents an error from the api.
type ErrorCode string

const (
	DatabaseError       ErrorCode = "databaseError"       // An internal problem with the database
	BadIDGeneration     ErrorCode = "badIdGeneration"     // Server failed to generate a valid api key, namespace id or pid
	InsufficientEntropy ErrorCode = "insufficientEntropy" // Insufficient entropy for api, namespace or pid generation

	BodyMissing      ErrorCode = "bodyMissing"      // request body was missing (but it was required)
	BodySizeExceeded ErrorCode = "bodySizeExceeded" // request body size limit exceeded
	BodyInvalidJSON  ErrorCode = "bodyInvalidJson"  // request body did not contain JSON, or it was not in the expected format

	InvalidQueryParameter ErrorCode = "invalidQueryParameter" // A query parameter that was sent was invalid

	ItemLimitExceeded ErrorCode = "itemLimitExceeded" // The number of items in the request exceeded the limit

	InvalidNamespaceID       ErrorCode = "invalidNamespaceId"       // An invalid namespace id was sent
	InvalidPID               ErrorCode = "invalidPid"               // An invalid pid was sent
	InvalidUsername          ErrorCode = "invalidUsername"          // An invalid username was sent
	InvalidAutocompleteQuery ErrorCode = "invalidAutocompleteQuery" // An invalid query was sent
	InvalidPassword          ErrorCode = "invalidPassword"          // An invalid password was sent
	InvalidBaseURI           ErrorCode = "invalidBaseUri"           // An invalid base URI was sent
	InvalidResourceURL       ErrorCode = "invalidResourceUrl"       // An invalid resource URL was sent

	NamespaceNotFound ErrorCode = "namespaceNotFound" // Namespace not found
	ResourceNotFound  ErrorCode = "resourceNotFound"  // Resource not found
	MountNotFound     ErrorCode = "mountNotFound"     // Mount not found

	RoleNotFound ErrorCode = "roleNotFound" // Role record not found
	InvalidRole  ErrorCode = "invalidRole"  // Role is not allowed for this operation

	InfoUnavailable ErrorCode = "infoUnavailable" // Info is unavailable (possibly for security reasons)

	UnavailableInAnonymousMode ErrorCode = "unavailableInAnonymousMode" // Route is not available in anonymous mode

	DuplicateUsername ErrorCode = "duplicateUsername" // Username is already in use
	UserNotFound      ErrorCode = "userNotFound"      // User not found
	KeyNotFound       ErrorCode = "keyNotFound"       // API key not found

	Unauthorized ErrorCode = "unauthorized" // Authentication is required but no valid credentials were provided
	Forbidden    ErrorCode = "forbidden"    // The authenticated user is not allowed to perform this action
)

type ErrorResponse struct {
	Error string `json:"error"`
}

// String returns the underlying error code as a string.
func (e ErrorCode) String() string {
	return string(e)
}

func (e ErrorCode) ToResponse() ErrorResponse {
	return ErrorResponse{Error: e.String()}
}

// HTTPCode returns the HTTP status code for the error.
//
// If an error code is invalid, it panics.
func (e ErrorCode) HTTPCode() int {
	switch e {
	case DatabaseError, BadIDGeneration:
		return http.StatusInternalServerError
	case InsufficientEntropy:
		return http.StatusServiceUnavailable
	case BodySizeExceeded:
		return http.StatusRequestEntityTooLarge
	case ItemLimitExceeded:
		return http.StatusUnprocessableEntity
	case DuplicateUsername:
		return http.StatusConflict
	case Unauthorized:
		return http.StatusUnauthorized
	case Forbidden:
		return http.StatusForbidden
	case InvalidQueryParameter, BodyMissing, BodyInvalidJSON,
		InvalidNamespaceID, InvalidPID, InvalidUsername, InvalidAutocompleteQuery, InvalidPassword, InvalidBaseURI,
		InvalidResourceURL, InvalidRole:
		return http.StatusBadRequest
	case NamespaceNotFound, ResourceNotFound, MountNotFound,
		RoleNotFound, InfoUnavailable, UnavailableInAnonymousMode,
		UserNotFound, KeyNotFound:
		return http.StatusNotFound
	default:
		panic("invalid error code")
	}
}

// WithErrorCode annotates err with the given [ErrorCode].
//
// The error message remains unchanged.
// The new error can be Unwrapped to get the underlying error.
func WithErrorCode(err error, code ErrorCode) error {
	return withCodeError{code: code, err: err}
}

// GetErrorCode extracts an [ErrorCode] that was added to the err using [WithErrorCode].
// If no [ErrorCode] was added, the second return value is false.
func GetErrorCode(err error) (ErrorCode, bool) {
	annotated, ok := errors.AsType[withCodeError](err)
	return annotated.code, ok
}

type withCodeError struct {
	code ErrorCode
	err  error
}

func (err withCodeError) Error() string {
	// we don't just call err.err.Error() here to avoid err.err == nil or panicking.
	// that's better handled by sprintf.
	return fmt.Sprintf("%s", err.err)
}

func (err withCodeError) Unwrap() error {
	return err.err
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
