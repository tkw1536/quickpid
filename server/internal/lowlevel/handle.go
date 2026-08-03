//spellchecker:words lowlevel
package lowlevel

//spellchecker:words errors slog http time github quickpid server internal credentials
import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/server/internal/credentials"
)

func (h *Handler) handle[T any](
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

		user, err := h.resolverCaller(r, s)
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

var (
	errInvalidAuthenticationCredentials = errors.New("invalid authentication credentials")
	errAuthenticationRequired           = errors.New("authentication required")
)

// resolverCaller resolves the caller of an API call according to the given scenario.
//
// It may return nil values when authentication is disabled, optional authentication is not supplied.
// Otherwise err === nil implies the user information is not nil.
//
// It can return errors annotated with:
//
// - [api.UnavailableInAnonymousMode] if the endpoint is unavailable in anonymous mode.
// - [api.Unauthorized] if the authentication credentials are invalid, or authentication is required but not supplied.
// - [api.DatabaseError] if the user found in the credentials cannot be loaded.
func (h *Handler) resolverCaller(r *http.Request, scenario scenario) (*api.ValidUserInfo, error) {
	// if we require auth mode, and we are in anonymous mode
	// this endpoint is unavailable.
	if scenario == requiredUser && h.auth.AnonymousMode() {
		return nil, api.WithErrorString(errUnavailableInAnonymousMode, api.UnavailableInAnonymousMode)
	}

	// if we said not to do any auth, don't load anything.
	if scenario == noAuthentication {
		return nil, nil
	}

	var (
		username api.ValidUsername
		err      error
	)

	creds := credentials.Parse(r.Header)
	switch creds.Kind() {
	case credentials.KindInvalid:
		return nil, api.WithErrorString(errInvalidAuthenticationCredentials, api.Unauthorized)
	case credentials.KindEmpty:
		if scenario != optionalUser {
			return nil, api.WithErrorString(errAuthenticationRequired, api.Unauthorized)
		}
		return nil, nil
	case credentials.KindBearer:
		username, err = h.auth.AuthenticateAPIKey(r.Context(), creds.BearerToken())
		if err != nil {
			return nil, api.WithErrorString(fmt.Errorf("invalid api key: %w", err), api.Unauthorized)
		}
	case credentials.KindBasic:
		basicUsername, basicPassword := creds.BasicAuth()
		username, err = h.auth.AuthenticatePassword(r.Context(), basicUsername, basicPassword)
		if err != nil {
			return nil, api.WithErrorString(fmt.Errorf("invalid username and password: %w", err), api.Unauthorized)
		}
	default:
		panic("never reached")
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
