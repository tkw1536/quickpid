//spellchecker:words lowlevel
package lowlevel

//spellchecker:words context slog http github bicpid
import (
	"context"
	"log/slog"
	"net/http"

	"github.com/tkw1536/bicpid/api"
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
	// Authenticate resolves a username from an API key.
	Authenticate(ctx context.Context, apiKey string) (string, error)

	// CurrentUser resolves the authenticated user from an API key.
	CurrentUser(ctx context.Context, apiKey string) (*api.UserInfo, error)
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
	impl func(http.ResponseWriter, *http.Request) (T, api.Error, error),
	successCode int,
	allowedErrors []api.Error,
) http.HandlerFunc {
	return handle(
		h,
		authNone(),
		func(w http.ResponseWriter, r *http.Request, _ *string, _ *api.UserInfo) (T, api.Error, error) {
			return impl(w, r)
		},
		successCode,
		allowedErrors,
	)
}

// HandleRequiredUsername wraps an endpoint that requires authentication and exposes the username.
func HandleRequiredUsername[T any](
	h *AuthHandler,
	impl func(http.ResponseWriter, *http.Request, string) (T, api.Error, error),
	successCode int,
	allowedErrors []api.Error,
) http.HandlerFunc {
	return handle(
		h,
		requiredUsernameAuth(),
		func(w http.ResponseWriter, r *http.Request, username *string, _ *api.UserInfo) (T, api.Error, error) {
			if username == nil {
				panic("never reached: required username authentication returned nil username")
			}
			return impl(w, r, *username)
		},
		successCode,
		allowedErrors,
	)
}

// HandleOptionalUsername wraps an endpoint that optionally authenticates and exposes the username when present.
func HandleOptionalUsername[T any](
	h *AuthHandler,
	impl func(http.ResponseWriter, *http.Request, *string) (T, api.Error, error),
	successCode int,
	allowedErrors []api.Error,
) http.HandlerFunc {
	return handle(
		h,
		optionalUsernameAuth(),
		func(w http.ResponseWriter, r *http.Request, username *string, _ *api.UserInfo) (T, api.Error, error) {
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
	impl func(http.ResponseWriter, *http.Request, *api.UserInfo) (T, api.Error, error),
	successCode int,
	allowedErrors []api.Error,
) http.HandlerFunc {
	return handle(
		h,
		requiredUserAuth(),
		func(w http.ResponseWriter, r *http.Request, _ *string, user *api.UserInfo) (T, api.Error, error) {
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
	impl func(http.ResponseWriter, *http.Request, *api.UserInfo) (T, api.Error, error),
	successCode int,
	allowedErrors []api.Error,
) http.HandlerFunc {
	return handle(
		h,
		optionalUserAuth(),
		func(w http.ResponseWriter, r *http.Request, _ *string, user *api.UserInfo) (T, api.Error, error) {
			return impl(w, r, user)
		},
		successCode,
		allowedErrors,
	)
}
