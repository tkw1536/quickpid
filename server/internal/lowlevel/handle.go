//spellchecker:words lowlevel
package lowlevel

//spellchecker:words errors slog http time github quickpid service
import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/service"
)

// handle executes the shared request flow used by all exported Handle* functions.
func handle[T any](
	h *AuthHandler,
	auth authConfig,
	impl func(http.ResponseWriter, *http.Request, *api.ValidUsername, *api.ValidUserInfo) (T, error),
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
func (h *AuthHandler) resolveAuth(r *http.Request, auth authConfig) (*api.ValidUsername, *api.ValidUserInfo, error) {
	if auth.requirement == authRequirementAuthMode && h.auth.AnonymousMode() {
		return nil, nil, api.WithErrorString(errUnavailableInAnonymousMode, api.UnavailableInAnonymousMode)
	}

	if auth.requirement == authRequirementNone {
		return nil, nil, nil
	}

	creds := readCredentials(r)
	if creds.invalid {
		return nil, nil, api.WithErrorString(errors.New("invalid authentication credentials"), api.Unauthorized)
	}
	if auth.requirement == authRequirementOptional && !creds.hasBearer() && !creds.hasBasic() {
		return nil, nil, nil
	}

	var (
		username api.ValidUsername
		err      error
	)
	switch {
	case creds.hasBearer():
		username, err = h.auth.AuthenticateAPIKey(r.Context(), creds.bearerToken)
		if service.IsUnauthorized(err) {
			return nil, nil, api.WithErrorString(fmt.Errorf("auth.AuthenticateAPIKey: %w", err), api.Unauthorized)
		}
		if err != nil {
			return nil, nil, api.WithErrorString(fmt.Errorf("auth.AuthenticateAPIKey: %w", err), api.DatabaseError)
		}
	case creds.hasBasic():
		username, err = h.auth.AuthenticatePassword(r.Context(), creds.basicUsername, creds.basicPassword)
		if service.IsUnauthorized(err) {
			return nil, nil, api.WithErrorString(fmt.Errorf("auth.AuthenticatePassword: %w", err), api.Unauthorized)
		}
		if err != nil {
			return nil, nil, api.WithErrorString(fmt.Errorf("auth.AuthenticatePassword: %w", err), api.DatabaseError)
		}
	default:
		return nil, nil, api.WithErrorString(errors.New("missing authentication credentials"), api.Unauthorized)
	}

	if !auth.loadUser {
		return &username, nil, nil
	}

	user, err := h.auth.LoadUser(r.Context(), username)
	if service.IsUnauthorized(err) {
		return nil, nil, api.WithErrorString(fmt.Errorf("auth.LoadUser: %w", err), api.Unauthorized)
	}
	if err != nil {
		return nil, nil, api.WithErrorString(fmt.Errorf("auth.LoadUser: %w", err), api.DatabaseError)
	}
	return &user.Username, &user, nil
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
