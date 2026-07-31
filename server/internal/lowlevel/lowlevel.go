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
// If the error is not nil, it should wrap an [api.ErrorString] value, which will be returned to the client.
// If there is no ErrorString, or it is not explicitly listed in the allowedErrors slice, this is considered an implementation error.
//
// Each of the methods is intended to support a different authentication scenario.
// - [Handler.Public] creates handlers that do not perform any authentication.
// - [Handler.Restricted] requires an authenticated user, and loads the whole user object from the database.
// - [Handler.Open] is like [Handler.Restricted], but also allows anonymous users.
//
// Note that depending on the authentication scenario, they introduce additional possible [api.ErrorString] values, which act
// as if they had been returned by the implementation function and thus need to be listed in the allowedErrors slice.
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
	//
	// Any non-nil error is considered to be an unauthorized user.
	AuthenticateAPIKey(ctx context.Context, apiKey string) (api.ValidUsername, error)

	// AuthenticatePassword resolves a username from a username / password pair.
	//
	// Any non-nil error is considered to be an unauthorized user.
	AuthenticatePassword(ctx context.Context, username string, password string) (api.ValidUsername, error)

	// LoadUser loads a full user object from a username.
	//
	// This method is only called after [AuthenticateAPIKey] or [AuthenticatePassword] have succeeded.
	// Hence any error is considered a database error, regardless of the actual underlying error.
	LoadUser(ctx context.Context, username api.ValidUsername) (api.ValidUserInfo, error)

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
