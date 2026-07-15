//spellchecker:words lowlevel
package lowlevel

//spellchecker:words errors slog http time github bicpid service
import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/tkw1536/bicpid/api"
	"github.com/tkw1536/bicpid/service"
)

// handle executes the shared request flow used by all exported Handle* functions.
func handle[T any](
	h *AuthHandler,
	auth authConfig,
	impl func(http.ResponseWriter, *http.Request, *string, *api.UserInfo) (T, error),
	successCode int,
	allowedErrors []api.ErrorString,
) http.HandlerFunc {
	errors := make(map[api.ErrorString]struct{}, len(allowedErrors))
	for _, err := range allowedErrors {
		errors[err] = struct{}{}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		username, user, err := h.resolveAuth(r, auth)
		if err != nil {
			duration := time.Since(start)
			h.writeHandledError(w, r, duration, err, errors)
			return
		}

		value, err := impl(w, r, username, user)
		duration := time.Since(start)

		if err != nil {
			h.writeHandledError(w, r, duration, err, errors)
			return
		}

		h.Log(r.Context(), r, duration, successCode)
		h.writeJSONResponse(w, r, successCode, value)
	}
}

var errUnavailableInAnonymousMode = errors.New("unavailable in anonymous mode")

func resolveAPIError(err error) api.ErrorString {
	if code, ok := api.GetErrorString(err); ok {
		return code
	}
	switch {
	case service.IsUnauthorized(err):
		return api.Unauthorized
	case service.IsForbidden(err):
		return api.Forbidden
	case service.IsUnavailableInAnonymousMode(err):
		return api.UnavailableInAnonymousMode
	default:
		return api.DatabaseError
	}
}

// resolveAuth resolves the caller according to the given auth configuration.
//
// It returns nil values when auth is disabled or optional auth is not supplied.
func (h *AuthHandler) resolveAuth(r *http.Request, auth authConfig) (*string, *api.UserInfo, error) {
	if auth.requirement == authRequirementAuthMode && h.auth.AnonymousMode() {
		return nil, nil, api.WithErrorString(errUnavailableInAnonymousMode, api.UnavailableInAnonymousMode)
	}

	if auth.requirement == authRequirementNone {
		return nil, nil, nil
	}

	token := readBearerToken(r)
	if auth.requirement == authRequirementOptional && token == "" {
		return nil, nil, nil
	}

	if auth.loadUser {
		user, err := h.auth.CurrentUser(r.Context(), token)
		if service.IsUnauthorized(err) {
			return nil, nil, api.WithErrorString(fmt.Errorf("auth.CurrentUser: %w", err), api.Unauthorized)
		}
		if err != nil {
			return nil, nil, api.WithErrorString(fmt.Errorf("auth.CurrentUser: %w", err), api.DatabaseError)
		}
		return &user.Username, user, nil
	}

	username, err := h.auth.Authenticate(r.Context(), token)
	if service.IsUnauthorized(err) {
		return nil, nil, api.WithErrorString(fmt.Errorf("auth.Authenticate: %w", err), api.Unauthorized)
	}
	if err != nil {
		return nil, nil, api.WithErrorString(fmt.Errorf("auth.Authenticate: %w", err), api.DatabaseError)
	}
	return &username, nil, nil
}

// writeHandledError validates that the error maps to a declared API error, logs it, and writes the JSON response.
func (h *AuthHandler) writeHandledError(
	w http.ResponseWriter,
	r *http.Request,
	duration time.Duration,
	err error,
	allowedErrors map[api.ErrorString]struct{},
) {
	specError := resolveAPIError(err)
	if _, ok := allowedErrors[specError]; !ok {
		panic("implementation error: unexpected error returned")
	}

	h.Log(
		r.Context(),
		r,
		duration,
		specError.HTTPCode(),
		slog.String("error", string(specError)),
		slog.Any("cause", err),
	)
	h.writeJSONResponse(w, r, specError.HTTPCode(), api.ErrorResponse{Error: specError})
}
