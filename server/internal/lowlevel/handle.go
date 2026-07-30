//spellchecker:words lowlevel
package lowlevel

//spellchecker:words errors slog http time github quickpid
import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/tkw1536/quickpid/api"
)

func handle[T any](
	h *Handler,
	s scenario,
	impl func(http.ResponseWriter, *http.Request, *api.ValidUserInfo) (T, error),
	successCode func(T) int,
	allowedErrors []api.ErrorString,
) http.HandlerFunc {
	errors := make(map[api.ErrorString]struct{}, len(allowedErrors))
	for _, err := range allowedErrors {
		errors[err] = struct{}{}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		user, err := h.resolveAuth(r, s)
		if err != nil {
			duration := time.Since(start)
			h.writeHandledError(w, r, duration, err, errors)
			return
		}

		value, err := impl(w, r, user)
		duration := time.Since(start)

		if err != nil {
			h.writeHandledError(w, r, duration, err, errors)
			return
		}

		status := successCode(value)
		h.Log(r.Context(), r, duration, status)
		h.writeJSONResponse(w, r, status, value)
	}
}

var errUnavailableInAnonymousMode = errors.New("unavailable in anonymous mode")

var errInvalidAuthenticationCredentials = errors.New("invalid authentication credentials")

// resolveAuth resolves the caller according to the given auth configuration.
//
// It returns nil values when auth is disabled or optional auth is not supplied.
func (h *Handler) resolveAuth(r *http.Request, scenario scenario) (*api.ValidUserInfo, error) {
	// if we require auth mode, and we are in anonymous mode
	// this endpoint is unavailable.
	if scenario == requiredUser && h.auth.AnonymousMode() {
		return nil, api.WithErrorString(errUnavailableInAnonymousMode, api.UnavailableInAnonymousMode)
	}

	// if we said not to do any auth, don't load anything.
	if scenario == noAuthentication {
		return nil, nil
	}

	// read the credentials from the request.
	creds := readCredentials(r)
	if creds.invalid {
		return nil, api.WithErrorString(errInvalidAuthenticationCredentials, api.Unauthorized)
	}
	if scenario == optionalUser && !creds.hasBearer() && !creds.hasBasic() {
		return nil, nil
	}

	var (
		username api.ValidUsername
		err      error
	)
	switch {
	case creds.hasBearer():
		username, err = h.auth.AuthenticateAPIKey(r.Context(), creds.bearerToken)
		if err != nil {
			return nil, api.WithErrorString(fmt.Errorf("invalid api key: %w", err), api.Unauthorized)
		}
	case creds.hasBasic():
		username, err = h.auth.AuthenticatePassword(r.Context(), creds.basicUsername, creds.basicPassword)
		if err != nil {
			return nil, api.WithErrorString(fmt.Errorf("invalid username and password: %w", err), api.Unauthorized)
		}
	default:
		return nil, api.WithErrorString(errInvalidAuthenticationCredentials, api.Unauthorized)
	}

	user, err := h.auth.LoadUser(r.Context(), username)
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("failed to load user object: %w", err), api.DatabaseError)
	}
	return &user, nil
}

// writeHandledError validates that the error maps to a declared API error, logs it, and writes the JSON response.
func (h *Handler) writeHandledError(
	w http.ResponseWriter,
	r *http.Request,
	duration time.Duration,
	err error,
	allowedErrors map[api.ErrorString]struct{},
) {
	specError, ok := api.GetErrorString(err)
	if !ok {
		panic("implementation error: did not carry an error string")
	}
	if _, ok := allowedErrors[specError]; !ok {
		panic("implementation error: unexpected error " + specError + " returned")
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
