//spellchecker:words lowlevel
package lowlevel

//spellchecker:words context errors slog github quickpid
import (
	"context"
	"errors"
	"log/slog"

	"github.com/tkw1536/quickpid/api"
)

//spellchecker:words marshallable

// Handler generates [http.HandlerFunc]s for quickpid-based functionality.
//
// The handler runs basic authentication, request logging as well as response serialization.
// A new handler can be created with [NewHandler], it requires database access using [AuthService].
// Afterwards, new [http.HandlerFunc]s can be created using the three methods on this struct.
//
// Each method takes an implementation function which accepts the user object (if any) as an argument.
// These represent the authenticated user (if any) for the request.
// The implementation functions must only access the user object using these variable, and should never access them directly.
// The implementation function should return two values:
//
// - a JSON marshallable result value of type T
// - an error value
//
// When the error is nil, the result of the implementation is interpreted as "successful".
// The result will always be marshaled into the response body as JSON.
// The status code of the response will be determined by the successCode function, which is typically [FixedStatusCode].
//
// If the error is not nil, it should wrap an [api.ErrorCode] value, which will be returned to the client.
// If there is no ErrorString, or it is not explicitly listed in the allowedErrors slice, this is considered an implementation error.
//
// Methods additionally may check specific [api.UserScope] or [api.NamespaceScope] values.
// If using a method that supports these scopes, they will be checked before the handler is ever called.
// If they fail, the return appropriate [api.ErrorCode] values.
type Handler struct {
	auth   AuthService
	logger *slog.Logger
}

// NewHandler creates a new [Handler].
func NewHandler(auth AuthService, logger *slog.Logger) *Handler {
	return &Handler{
		auth:   auth,
		logger: logger,
	}
}

// AuthService provides the authentication operations needed by [Handler].
type AuthService interface {
	// AuthenticateAPIKey resolves a username from an API key.
	// It returns the username, and the key used to authenticate it.
	//
	// Any non-nil error is considered to be an unauthorized user.
	AuthenticateAPIKey(ctx context.Context, apiKey string) (api.ValidUsername, api.APIKeyInfo, error)

	// AuthenticatePassword resolves a username from a username / password pair.
	//
	// Any non-nil error is considered to be an unauthorized user.
	AuthenticatePassword(ctx context.Context, username string, password string) (api.ValidUsername, error)

	// LoadUser loads a full user object from a username.
	//
	// This method is only called after [AuthenticateAPIKey] or [AuthenticatePassword] have succeeded.
	// Hence any error is considered a database error, regardless of the actual underlying error.
	LoadUser(ctx context.Context, username api.ValidUsername) (api.ValidUserInfo, error)

	// ExplicitNamespaceRole returns the user's explicit role in a namespace for scope checks.
	// Returns [api.RoleNone] when no role is stored.
	// Errors are annotated with [api.NamespaceNotFound] or [api.DatabaseError].
	ExplicitNamespaceRole(ctx context.Context, namespace api.ValidNamespaceID, username api.ValidUsername) (api.Role, error)

	// AnonymousMode reports whether the server is in anonymous mode.
	AnonymousMode() bool
}

// ErrorUnauthorized is the error returned when authentication credentials are invalid.
var ErrUnauthorized = errors.New("invalid authentication credentials")

// FixedStatusCode returns a function that always returns the same status code.
func FixedStatusCode[T any](status int) func(T) int {
	return func(result T) int {
		return status
	}
}
