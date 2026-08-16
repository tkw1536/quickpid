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

// Public returns a new handler that does not consider any authentication (not even validating for correctness).
//
// When successful, the handler returns the result of the provided implementation.
// When an error occurs, the handler returns the appropriate error code.
//
// successCode is a function that converts the result of the implementation into a HTTP status code.
// allowedErrors is a list of error codes that are allowed to be returned by the implementation; if an error is returned that is not in this list, the handler will panic.
func (h *Handler) Public[T any](
	impl func(http.ResponseWriter, *http.Request) (T, error),
	successCode func(T) int,
	allowedErrors []api.ErrorCode,
) http.HandlerFunc {
	return h.handle(
		false,
		func(w http.ResponseWriter, r *http.Request, _ *api.Caller) (T, error) {
			return impl(w, r)
		},
		successCode,
		allowedErrors,
	)
}

// UserScope returns a handler that requires the provided user scope to be fulfilled.
//
// When the server is in anonymous mode, the handler returns [api.UnavailableInAnonymousMode].
// When in authenticated mode, the handler can return [api.Unauthorized] and [api.DatabaseError].
//
// When successful, the handler returns the result of the provided implementation.
// When the scope is not fulfilled, the handler returns an error derived from checking the appropriate scope.
//
// successCode is a function that converts the result of the implementation into a HTTP status code.
// allowedErrors is a list of error codes that are allowed to be returned by the implementation; if an error is returned that is not in this list, the handler will panic.
func (h *Handler) UserScope[T any](
	scope api.UserScope,
	impl func(http.ResponseWriter, *http.Request, *api.Caller) (T, error),
	successCode func(T) int,
	allowedErrors []api.ErrorCode,
) http.HandlerFunc {
	return h.dynamicUserScope(
		func(r *http.Request, user *api.Caller) (struct{}, api.UserScope, error) {
			return struct{}{}, scope, nil
		},
		func(w http.ResponseWriter, r *http.Request, user *api.Caller, _ struct{}) (T, error) {
			return impl(w, r, user)
		}, successCode, allowedErrors)
}

// dynamicUserScope is like [UserScope], except that the scope may be dynamically determined by impl.
// impl *must* call the provided check function exactly once per invocation, or the handler will panic.
func (h *Handler) dynamicUserScope[T, S any](
	scope func(r *http.Request, user *api.Caller) (S, api.UserScope, error),
	impl func(http.ResponseWriter, *http.Request, *api.Caller, S) (T, error),
	successCode func(T) int,
	allowedErrors []api.ErrorCode,
) http.HandlerFunc {
	return h.handle(
		true,
		func(w http.ResponseWriter, r *http.Request, user *api.Caller) (T, error) {
			v, s, err := scope(r, user)
			if err != nil {
				var zero T
				return zero, fmt.Errorf("scope evaluation failed: %w", err)
			}

			if err := h.CheckUserScope(r, user, s); err != nil {
				var zero T
				return zero, fmt.Errorf("scope evaluation failed: %w", err)
			}

			result, err := impl(w, r, user, v)
			if err != nil {
				var zero T
				return zero, fmt.Errorf("implementation failed: %w", err)
			}

			return result, nil
		},
		successCode,
		allowedErrors,
	)
}

// CheckUserScope evaluates if the given action is available for the given user.
//
// When the action is not available, an error of one of the following [api.ErrorCode]s is returned:
//
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.UnavailableInAnonymousMode]
// - [api.DatabaseError].
func (h *Handler) CheckUserScope(r *http.Request, user *api.Caller, scope api.UserScope) error {
	if h.auth.AnonymousMode() {
		if err := api.EvaluateAnonymousModeUserScope(scope); err != nil {
			return fmt.Errorf("scope %q not fulfilled in anonymous mode: %w", scope, err)
		}
		return nil
	}
	if err := api.EvaluateUserScope(user, scope); err != nil {
		return fmt.Errorf("scope %q not fulfilled for user %q: %w", scope, user.Username(), err)
	}
	return nil
}

// NamespaceScope returns a handler that parses the namespace path parameter and requires the provided namespace scope.
//
// When the server is in anonymous mode, the handler returns [api.UnavailableInAnonymousMode].
// When in authenticated mode, the handler can return [api.Unauthorized], [api.Forbidden], [api.NamespaceNotFound], and [api.DatabaseError].
// Invalid namespace path values return [api.InvalidNamespaceID].
//
// When successful, the handler returns the result of the provided implementation.
// When the scope is not fulfilled, the handler returns an error derived from checking the appropriate scope.
// /
// successCode is a function that converts the result of the implementation into a HTTP status code.
// allowedErrors is a list of error codes that are allowed to be returned by the implementation; if an error is returned that is not in this list, the handler will panic.
func (h *Handler) NamespaceScope[T any](
	scope api.NamespaceScope,
	impl func(http.ResponseWriter, *http.Request, *api.Caller, api.ValidNamespaceID) (T, error),
	successCode func(T) int,
	allowedErrors []api.ErrorCode,
) http.HandlerFunc {
	return h.DynamicNamespaceScope(
		func(r *http.Request, user *api.Caller) (struct{}, api.NamespaceScope, error) {
			return struct{}{}, scope, nil
		},
		func(w http.ResponseWriter, r *http.Request, user *api.Caller, namespace api.ValidNamespaceID, _ struct{}) (T, error) {
			return impl(w, r, user, namespace)
		}, successCode, allowedErrors)
}

// DynamicNamespaceScope is like [NamespaceScope], except that the scope may be dynamically determined by impl.
//
// impl *must* call the provided check function exactly once on the success path.
// Returning an error before calling check (for example path-parameter validation) is allowed;
// returning successfully without calling check, or calling check more than once, panics.
func (h *Handler) DynamicNamespaceScope[T, S any](
	scope func(r *http.Request, user *api.Caller) (S, api.NamespaceScope, error),
	impl func(http.ResponseWriter, *http.Request, *api.Caller, api.ValidNamespaceID, S) (T, error),
	successCode func(T) int,
	allowedErrors []api.ErrorCode,
) http.HandlerFunc {
	return h.handle(
		true,
		func(w http.ResponseWriter, r *http.Request, user *api.Caller) (T, error) {
			v, s, err := scope(r, user)
			if err != nil {
				var zero T
				return zero, fmt.Errorf("scope evaluation failed: %w", err)
			}

			namespace, err := readNamespaceParam(r)
			if err != nil {
				var zero T
				return zero, fmt.Errorf("namespace parsing failed: %w", err)
			}

			if err := h.CheckNamespaceScope(r, namespace, user, s); err != nil {
				var zero T
				return zero, fmt.Errorf("namespace evaluation failed: %w", err)
			}

			result, err := impl(w, r, user, namespace, v)
			if err != nil {
				var zero T
				return zero, fmt.Errorf("implementation failed: %w", err)
			}

			return result, nil
		},
		successCode,
		allowedErrors,
	)
}

// readNamespaceParam gets the namespace from the request path.
//
// It can return the following errors:
//
// - [api.InvalidNamespaceID].
func readNamespaceParam(r *http.Request) (api.ValidNamespaceID, error) {
	namespace, err := api.NewNamespaceID(r.PathValue("namespace"))
	if err != nil {
		return api.ValidNamespaceID{}, api.WithErrorCode(fmt.Errorf("failed to parse namespace id: %w", err), api.InvalidNamespaceID)
	}
	return namespace, nil
}

// CheckNamespaceScope evaluates if the given action is available for the given namespace.
//
// When the action is not available, an error of one of the following [api.ErrorCode]s is returned:
//
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.UnavailableInAnonymousMode]
// - [api.NamespaceNotFound]
// - [api.DatabaseError].
func (h *Handler) CheckNamespaceScope(r *http.Request, namespace api.ValidNamespaceID, caller *api.Caller, scope api.NamespaceScope) error {
	if h.auth.AnonymousMode() {
		if err := api.EvaluateAnonymousModeNamespaceScope(namespace, scope); err != nil {
			return fmt.Errorf("scope %q not fulfilled for namespace %q: %w", scope, namespace.String(), err)
		}
		return nil
	}

	role := api.RoleNone
	if caller != nil {
		var err error
		role, err = h.auth.ExplicitNamespaceRole(r.Context(), namespace, caller.Username())
		if err != nil {
			return fmt.Errorf("scope %q not fulfilled for namespace %q: %w", scope, namespace.String(), err)
		}
	}

	if err := api.EvaluateNamespaceScope(namespace, role, caller, scope); err != nil {
		return fmt.Errorf("scope %q not fulfilled for namespace %q: %w", scope, namespace.String(), err)
	}
	return nil
}

// handle implements the common logic for all lowlevel handlers.
func (h *Handler) handle[T any](
	auth bool,
	impl func(http.ResponseWriter, *http.Request, *api.Caller) (T, error),
	successCode func(T) int,
	allowedErrors []api.ErrorCode,
) http.HandlerFunc {
	errors := make(map[api.ErrorCode]struct{}, len(allowedErrors))
	for _, err := range allowedErrors {
		errors[err] = struct{}{}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		var user *api.Caller
		if auth {
			var err error
			user, err = h.resolveCaller(r)
			if err != nil {
				duration := time.Since(start)
				h.writeHandledError(w, r, duration, err, errors)
				return
			}
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

var (
	errInvalidAuthenticationCredentials = errors.New("invalid authentication credentials")
)

// resolveCaller resolves the caller of an API call.
//
// It can return errors annotated with:
//
//   - [api.UnavailableInAnonymousMode] if the endpoint is unavailable in anonymous mode.
//   - [api.Unauthorized] if the authentication credentials are invalid, authentication is required but not supplied,
//     or impersonation fails (non-superuser, invalid or missing target, or multiple impersonate headers).
//   - [api.DatabaseError] if the authenticated or impersonated user cannot be loaded for a database reason.
func (h *Handler) resolveCaller(r *http.Request) (*api.Caller, error) {
	var (
		username api.ValidUsername
		method   api.AuthenticationMethod
		err      error
	)

	creds := credentials.Parse(r.Header)
	switch creds.Kind() {
	case credentials.KindInvalid:
		return nil, api.WithErrorCode(errInvalidAuthenticationCredentials, api.Unauthorized)
	case credentials.KindEmpty:
		return nil, nil
	case credentials.KindBearer:
		var key api.APIKeyInfo
		username, key, err = h.auth.AuthenticateAPIKey(r.Context(), creds.BearerToken())
		if err != nil {
			return nil, api.WithErrorCode(fmt.Errorf("invalid api key: %w", err), api.Unauthorized)
		}
		method = api.TokenAuthentication{
			Token: key,
		}
	case credentials.KindBasic:
		basicUsername, basicPassword := creds.BasicAuth()
		username, err = h.auth.AuthenticatePassword(r.Context(), basicUsername, basicPassword)
		if err != nil {
			return nil, api.WithErrorCode(fmt.Errorf("invalid username and password: %w", err), api.Unauthorized)
		}
		method = api.BasicAuthentication{}
	default:
		panic("never reached")
	}

	user, err := h.auth.LoadUser(r.Context(), username)
	if err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("failed to load user object: %w", err), api.DatabaseError)
	}
	return h.resolveImpersonatedUser(r, user.Caller(method))
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
func (h *Handler) resolveImpersonatedUser(r *http.Request, caller api.Caller) (*api.Caller, error) {
	// get the impersonate header, and handle cases of no header and multiple headers.
	headers := r.Header.Values(api.ImpersonateHeader)
	switch len(headers) {
	case 0:
		return &caller, nil
	case 1: /* see below */
	default:
		return nil, api.WithErrorCode(errMultipleImpersonateHeaders, api.Unauthorized)
	}

	if err := api.EvaluateUserScope(&caller, api.ScopeImpersonate); err != nil {
		return nil, api.WithErrorCode(fmt.Errorf("%w: %w", errNotAllowedToImpersonate, err), api.Unauthorized)
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

	return new(
		user.Caller(
			api.Impersonation{
				User:   username,
				Reason: caller.Method(),
			}),
	), nil
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
