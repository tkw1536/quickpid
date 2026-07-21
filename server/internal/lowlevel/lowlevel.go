//spellchecker:words lowlevel
package lowlevel

//spellchecker:words context slog http github quickpid
import (
	"context"
	"log/slog"
	"net/http"

	"github.com/tkw1536/quickpid/api"
)

// AuthHandler executes common HTTP request handling for authenticated and unauthenticated endpoints.
//
// It owns authentication, request logging, response serialization, and allowed-error enforcement.
type AuthHandler struct {
	auth   AuthService
	logger *slog.Logger
}

// AuthService provides the authentication operations needed by [AuthHandler].
type AuthService interface {
	// AuthenticateAPIKey resolves a username from an API key.
	AuthenticateAPIKey(ctx context.Context, apiKey string) (api.ValidUsername, error)

	// AuthenticatePassword resolves a username from a username / password pair.
	AuthenticatePassword(ctx context.Context, username string, password string) (api.ValidUsername, error)

	// LoadUser loads a full user object from a username.
	LoadUser(ctx context.Context, username api.ValidUsername) (api.ValidUserInfo, error)

	// AnonymousMode reports whether authentication is disabled for resolver operations.
	AnonymousMode() bool
}

// NewAuthHandler creates a new [AuthHandler].
func NewAuthHandler(auth AuthService, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{
		auth:   auth,
		logger: logger,
	}
}

// HandleNoAuth wraps an endpoint that does not perform authentication.
func HandleNoAuth[T any](
	h *AuthHandler,
	impl func(http.ResponseWriter, *http.Request) (T, error),
	successCode int,
	allowedErrors []api.ErrorString,
) http.HandlerFunc {
	return handle(
		h,
		authNone(),
		func(w http.ResponseWriter, r *http.Request, _ *api.ValidUsername, _ *api.ValidUserInfo) (T, error) {
			return impl(w, r)
		},
		successCode,
		allowedErrors,
	)
}

// HandleRequiredUsername wraps an endpoint that requires authentication and exposes the username.
func HandleRequiredUsername[T any](
	h *AuthHandler,
	impl func(http.ResponseWriter, *http.Request, *api.ValidUsername) (T, error),
	successCode int,
	allowedErrors []api.ErrorString,
) http.HandlerFunc {
	return handle(
		h,
		requiredUsernameAuth(),
		func(w http.ResponseWriter, r *http.Request, username *api.ValidUsername, _ *api.ValidUserInfo) (T, error) {
			if username == nil {
				panic("never reached: required username authentication returned nil username")
			}
			return impl(w, r, username)
		},
		successCode,
		allowedErrors,
	)
}

// HandleOptionalUsername wraps an endpoint that optionally authenticates and exposes the username when present.
func HandleOptionalUsername[T any](
	h *AuthHandler,
	impl func(http.ResponseWriter, *http.Request, *api.ValidUsername) (T, error),
	successCode int,
	allowedErrors []api.ErrorString,
) http.HandlerFunc {
	return handle(
		h,
		optionalUsernameAuth(),
		func(w http.ResponseWriter, r *http.Request, username *api.ValidUsername, _ *api.ValidUserInfo) (T, error) {
			return impl(w, r, username)
		},
		successCode,
		allowedErrors,
	)
}

// HandleRequiredUser wraps an endpoint that requires authentication and exposes the authenticated user.
//
// The provided user pointer is guaranteed to be non-nil.
func HandleRequiredUser[T any](
	h *AuthHandler,
	impl func(http.ResponseWriter, *http.Request, *api.ValidUserInfo) (T, error),
	successCode int,
	allowedErrors []api.ErrorString,
) http.HandlerFunc {
	return handle(
		h,
		requiredUserAuth(),
		func(w http.ResponseWriter, r *http.Request, _ *api.ValidUsername, user *api.ValidUserInfo) (T, error) {
			if user == nil {
				panic("never reached: required user authentication returned nil user")
			}
			return impl(w, r, user)
		},
		successCode,
		allowedErrors,
	)
}

// HandleRequiredUserInAuthMode wraps an auth-management endpoint that requires authentication
// when the server is not in anonymous mode.
func HandleRequiredUserInAuthMode[T any](
	h *AuthHandler,
	impl func(http.ResponseWriter, *http.Request, *api.ValidUserInfo) (T, error),
	successCode int,
	allowedErrors []api.ErrorString,
) http.HandlerFunc {
	return handle(
		h,
		requiredUserAuthManagement(),
		func(w http.ResponseWriter, r *http.Request, _ *api.ValidUsername, user *api.ValidUserInfo) (T, error) {
			if user == nil {
				panic("never reached: required user authentication returned nil user")
			}
			return impl(w, r, user)
		},
		successCode,
		allowedErrors,
	)
}

// HandleOptionalUser wraps an endpoint that optionally authenticates and exposes the authenticated user when present.
func HandleOptionalUser[T any](
	h *AuthHandler,
	impl func(http.ResponseWriter, *http.Request, *api.ValidUserInfo) (T, error),
	successCode int,
	allowedErrors []api.ErrorString,
) http.HandlerFunc {
	return handle(
		h,
		optionalUserAuth(),
		func(w http.ResponseWriter, r *http.Request, _ *api.ValidUsername, user *api.ValidUserInfo) (T, error) {
			return impl(w, r, user)
		},
		successCode,
		allowedErrors,
	)
}
