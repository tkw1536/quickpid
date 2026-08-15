//spellchecker:words lowlevel
package lowlevel

//spellchecker:words http sync github quickpid scopes
import (
	"fmt"
	"net/http"
	"sync"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/scopes"
)

// scenario describes an authentication scenario.
type scenario uint8

const (
	noAuthentication scenario = iota
	optionalUser
	requiredUser
)

// Public returns a handler that ignores any authentication (not even validating for correctness).
func (h *Handler) Public[T any](
	impl func(http.ResponseWriter, *http.Request) (T, error),
	successCode func(T) int,
	allowedErrors []api.ErrorCode,
) http.HandlerFunc {
	return h.handle(
		noAuthentication,
		func(w http.ResponseWriter, r *http.Request, _ *api.Caller) (T, error) {
			return impl(w, r)
		},
		successCode,
		allowedErrors,
	)
}

// Restricted returns a handler that requires a signed-in user.
//
// When the server is in anonymous mode, the handler returns [api.UnavailableInAnonymousMode].
// When in authenticated mode, the handler can return [api.Unauthorized] and [api.DatabaseError].
func (h *Handler) Restricted[T any](
	impl func(http.ResponseWriter, *http.Request, api.Caller) (T, error),
	successCode func(T) int,
	allowedErrors []api.ErrorCode,
) http.HandlerFunc {
	return h.handle(
		requiredUser,
		func(w http.ResponseWriter, r *http.Request, user *api.Caller) (T, error) {
			if user == nil {
				panic("never reached: required user authentication returned nil user")
			}
			return impl(w, r, *user)
		},
		successCode,
		allowedErrors,
	)
}

// Open returns a handler that optionally requires a signed-in user.
//
// When in anonymous mode, the handler returns no additional errors.
// When in authenticated mode, the handler can return [api.Unauthorized] and [api.DatabaseError].
func (h *Handler) Open[T any](
	impl func(http.ResponseWriter, *http.Request, *api.Caller) (T, error),
	successCode func(T) int,
	allowedErrors []api.ErrorCode,
) http.HandlerFunc {
	return h.handle(
		optionalUser,
		impl,
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
func (h *Handler) UserScope[T any](
	scope scopes.UserScope,
	impl func(http.ResponseWriter, *http.Request, *api.Caller) (T, error),
	successCode func(T) int,
	allowedErrors []api.ErrorCode,
) http.HandlerFunc {
	return h.DynamicUserScope(func(w http.ResponseWriter, r *http.Request, user *api.Caller, check func(scopes.UserScope) error) (T, error) {
		if err := check(scope); err != nil {
			var zero T
			return zero, err
		}
		return impl(w, r, user)
	}, successCode, allowedErrors)
}

// DynamicUserScope is like [UserScope], except that the scope may be dynamically determined by impl.
// impl *must* call the provided check function exactly once per invocation, or the handler will panic.
func (h *Handler) DynamicUserScope[T any](
	impl func(http.ResponseWriter, *http.Request, *api.Caller, func(scopes.UserScope) error) (T, error),
	successCode func(T) int,
	allowedErrors []api.ErrorCode,
) http.HandlerFunc {
	return h.handle(
		optionalUser,
		func(w http.ResponseWriter, r *http.Request, user *api.Caller) (T, error) {
			var (
				called bool
				m      sync.Mutex
			)

			result, err := impl(w, r, user, func(scope scopes.UserScope) error {
				m.Lock()
				defer m.Unlock()

				if called {
					panic("DynamicUserScope: check function called multiple times")
				}
				called = true

				if h.auth.AnonymousMode() {
					return scopes.EvaluateAnonymousModeUserScope(scope)
				}
				return scopes.EvaluateUserScope(user, scope)
			})

			if !called {
				panic("DynamicUserScope: check function not called")
			}

			if err != nil {
				var zero T
				return zero, fmt.Errorf("DynamicUserScope: %w", err)
			}

			return result, nil
		},
		successCode,
		allowedErrors,
	)
}
