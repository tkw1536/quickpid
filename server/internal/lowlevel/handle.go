//spellchecker:words lowlevel
package lowlevel

//spellchecker:words errors slog http strings time github quickpid server internal credentials
import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/server/internal/credentials"
)

func (h *Handler) handle[T any](
	s scenario,
	impl func(http.ResponseWriter, *http.Request, *api.ValidUserInfo) (T, error),
	successCode func(T) int,
	allowedErrors []api.ErrorCode,
) http.HandlerFunc {
	errors := make(map[api.ErrorCode]struct{}, len(allowedErrors))
	for _, err := range allowedErrors {
		errors[err] = struct{}{}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		user, err := h.resolveCaller(r, s)
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

// resolveCaller resolves the caller of an API call according to the given scenario.
//
// It may return nil values when authentication is disabled, optional authentication is not supplied.
// Otherwise err === nil implies the user information is not nil.
//
// It can return errors annotated with:
//
//   - [api.UnavailableInAnonymousMode] if the endpoint is unavailable in anonymous mode.
//   - [api.Unauthorized] if the authentication credentials are invalid, authentication is required but not supplied,
//     or impersonation fails (non-superuser, invalid or missing target, or multiple impersonate headers).
//   - [api.DatabaseError] if the authenticated or impersonated user cannot be loaded for a database reason.
func (h *Handler) resolveCaller(r *http.Request, scenario scenario) (*api.ValidUserInfo, error) {
	// if we require auth mode, and we are in anonymous mode
	// this endpoint is unavailable.
	if scenario == requiredUser && h.auth.AnonymousMode() {
		return nil, api.WithErrorCode(errUnavailableInAnonymousMode, api.UnavailableInAnonymousMode)
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
		return nil, api.WithErrorCode(errInvalidAuthenticationCredentials, api.Unauthorized)
	case credentials.KindEmpty:
		if scenario != optionalUser {
			return nil, api.WithErrorCode(errAuthenticationRequired, api.Unauthorized)
		}
		return nil, nil
	case credentials.KindBearer:
		username, err = h.auth.AuthenticateAPIKey(r.Context(), creds.BearerToken())
		if err != nil {
			return nil, api.WithErrorCode(fmt.Errorf("invalid api key: %w", err), api.Unauthorized)
		}
	case credentials.KindBasic:
		basicUsername, basicPassword := creds.BasicAuth()
		username, err = h.auth.AuthenticatePassword(r.Context(), basicUsername, basicPassword)
		if err != nil {
			return nil, api.WithErrorCode(fmt.Errorf("invalid username and password: %w", err), api.Unauthorized)
		}
	default:
		panic("never reached")
	}

	user, err := h.auth.LoadUser(r.Context(), username)
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("failed to load user object: %w", err), api.DatabaseError)
	}

	return h.resolveImpersonatedUser(r, user)
}

var (
	errMultipleImpersonateHeaders = errors.New("multiple impersonate headers")
	errNotAllowedToImpersonate    = errors.New("only superusers can impersonate other users")
)

// resolveImpersonatedUser returns the effective caller for a request.
//
// When X-Impersonate-User is absent, it returns caller unchanged.
// When present, only a superuser may impersonate; the impersonated user is loaded and returned.
//
// It can return errors annotated with:
//
//   - [api.Unauthorized] if multiple impersonate headers are sent, the caller is not a superuser,
//     the impersonated username is invalid, or the impersonated user does not exist.
//   - [api.DatabaseError] if loading the impersonated user fails for a reason other than not found.
func (h *Handler) resolveImpersonatedUser(r *http.Request, caller api.ValidUserInfo) (*api.ValidUserInfo, error) {
	// get the impersonate header, and handle cases of no header and multiple headers.
	headers := r.Header.Values(api.ImpersonateHeader)
	switch len(headers) {
	case 0:
		return &caller, nil
	case 1: /* see below */
	default:
		return nil, api.WithErrorCode(errMultipleImpersonateHeaders, api.Unauthorized)
	}

	if !caller.Superuser {
		return nil, api.WithErrorCode(errNotAllowedToImpersonate, api.Unauthorized)
	}

	username, err := api.NewUsername(strings.TrimSpace(headers[0]))
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("invalid impersonated username: %w", err), api.Unauthorized)
	}

	user, err := h.auth.LoadUser(r.Context(), username)
	if err != nil {
		if code, ok := api.GetErrorCode(err); ok && code == api.UserNotFound {
			return nil, api.WithErrorCode(fmt.Errorf("impersonated user not found: %w", err), api.Unauthorized)
		}
		return nil, api.WithErrorCode(fmt.Errorf("failed to load impersonated user object: %w", err), api.DatabaseError)
	}

	return &user, nil
}

// writeHandledError validates that the error maps to a declared API error, logs it, and writes the JSON response.
func (h *Handler) writeHandledError(
	w http.ResponseWriter,
	r *http.Request,
	duration time.Duration,
	err error,
	allowedErrors map[api.ErrorCode]struct{},
) {
	code, ok := api.GetErrorCode(err)
	if !ok {
		panic("implementation error: did not carry an error string")
	}
	if _, ok := allowedErrors[code]; !ok {
		panic("implementation error: unexpected error " + code.String() + " returned")
	}

	h.Log(
		r.Context(),
		r,
		duration,
		code.HTTPCode(),
		slog.String("error", code.String()),
		slog.Any("cause", err),
	)
	h.writeJSONResponse(w, r, code.HTTPCode(), code.ToResponse())
}
