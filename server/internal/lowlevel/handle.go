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
	impl func(http.ResponseWriter, *http.Request, *string, *api.UserInfo) (T, api.Error, error),
	successCode int,
	allowedErrors []api.Error,
) http.HandlerFunc {
	errors := make(map[api.Error]struct{}, len(allowedErrors))
	for _, err := range allowedErrors {
		errors[err] = struct{}{}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		username, user, specError, err := h.resolveAuth(r, auth)
		if err != nil {
			duration := time.Since(start)
			h.writeHandledError(w, r, duration, specError, err, errors)
			return
		}

		value, specError, err := impl(w, r, username, user)
		duration := time.Since(start)

		if err != nil {
			switch {
			case service.IsUnauthorized(err):
				h.writeHandledError(w, r, duration, api.Unauthorized, err, errors)
			case service.IsForbidden(err):
				h.writeHandledError(w, r, duration, api.Forbidden, err, errors)
			default:
				if specError == "" {
					specError = api.DatabaseError
				}
				h.writeHandledError(w, r, duration, specError, err, errors)
			}
			return
		}

		if specError != "" {
			panic("never reached: specError != \"\", but err == nil")
		}

		h.Log(r.Context(), r, duration, successCode)
		writeJSONResponse(w, successCode, value)
	}
}

var errUnavailableInAnonymousMode = errors.New("unavailable in anonymous mode")

// resolveAuth resolves the caller according to the given auth configuration.
//
// It returns nil values when auth is disabled or optional auth is not supplied.
func (h *AuthHandler) resolveAuth(r *http.Request, auth authConfig) (*string, *api.UserInfo, api.Error, error) {
	if auth.requirement == authRequirementAuthMode && h.auth.AnonymousMode() {
		return nil, nil, api.UnavailableInAnonymousMode, errUnavailableInAnonymousMode
	}

	if auth.requirement == authRequirementNone {
		return nil, nil, "", nil
	}

	token := readBearerToken(r)
	if auth.requirement == authRequirementOptional && token == "" {
		return nil, nil, "", nil
	}

	if auth.loadUser {
		user, err := h.auth.CurrentUser(r.Context(), token)
		if service.IsUnauthorized(err) {
			return nil, nil, api.Unauthorized, fmt.Errorf("auth.CurrentUser: %w", err)
		}
		if err != nil {
			return nil, nil, api.DatabaseError, fmt.Errorf("auth.CurrentUser: %w", err)
		}
		return &user.Username, user, "", nil
	}

	username, err := h.auth.Authenticate(r.Context(), token)
	if service.IsUnauthorized(err) {
		return nil, nil, api.Unauthorized, fmt.Errorf("auth.Authenticate: %w", err)
	}
	if err != nil {
		return nil, nil, api.DatabaseError, fmt.Errorf("auth.Authenticate: %w", err)
	}
	return &username, nil, "", nil
}

// writeHandledError validates that specError is declared in allowedErrors, logs it, and writes the JSON response.
func (h *AuthHandler) writeHandledError(
	w http.ResponseWriter,
	r *http.Request,
	duration time.Duration,
	specError api.Error,
	err error,
	allowedErrors map[api.Error]struct{},
) {
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
	writeJSONResponse(w, specError.HTTPCode(), api.ErrorResponse{Error: string(specError)})
}
