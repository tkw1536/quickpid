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
	return h.dynamicUserScope(func(w http.ResponseWriter, r *http.Request, user *api.Caller, check func(scopes.UserScope) error) (T, error) {
		if err := check(scope); err != nil {
			var zero T
			return zero, err
		}
		return impl(w, r, user)
	}, successCode, allowedErrors)
}

// dynamicUserScope is like [UserScope], except that the scope may be dynamically determined by impl.
// impl *must* call the provided check function exactly once per invocation, or the handler will panic.
func (h *Handler) dynamicUserScope[T any](
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

// NamespaceScope returns a handler that parses the namespace path parameter and requires the provided namespace scope.
//
// When the server is in anonymous mode, the handler returns [api.UnavailableInAnonymousMode].
// When in authenticated mode, the handler can return [api.Unauthorized], [api.Forbidden], [api.NamespaceNotFound], and [api.DatabaseError].
// Invalid namespace path values return [api.InvalidNamespaceID].
//
// When successful, the handler returns the result of the provided implementation.
// When the scope is not fulfilled, the handler returns an error derived from checking the appropriate scope.
func (h *Handler) NamespaceScope[T any](
	scope scopes.NamespaceScope,
	impl func(http.ResponseWriter, *http.Request, *api.Caller, api.ValidNamespaceID) (T, error),
	successCode func(T) int,
	allowedErrors []api.ErrorCode,
) http.HandlerFunc {
	return h.DynamicNamespaceScope(func(w http.ResponseWriter, r *http.Request, user *api.Caller, namespace api.ValidNamespaceID, check func(scopes.NamespaceScope) error) (T, error) {
		if err := check(scope); err != nil {
			var zero T
			return zero, err
		}
		return impl(w, r, user, namespace)
	}, successCode, allowedErrors)
}

// DynamicNamespaceScope is like [NamespaceScope], except that the scope may be dynamically determined by impl.
//
// impl *must* call the provided check function exactly once on the success path.
// Returning an error before calling check (for example path-parameter validation) is allowed;
// returning successfully without calling check, or calling check more than once, panics.
func (h *Handler) DynamicNamespaceScope[T any](
	impl func(http.ResponseWriter, *http.Request, *api.Caller, api.ValidNamespaceID, func(scopes.NamespaceScope) error) (T, error),
	successCode func(T) int,
	allowedErrors []api.ErrorCode,
) http.HandlerFunc {
	return h.handle(
		optionalUser,
		func(w http.ResponseWriter, r *http.Request, user *api.Caller) (T, error) {
			namespace, err := readNamespaceParam(r)
			if err != nil {
				var zero T
				return zero, err
			}

			var (
				called bool
				m      sync.Mutex
			)

			result, err := impl(w, r, user, namespace, func(scope scopes.NamespaceScope) error {
				m.Lock()
				defer m.Unlock()

				if called {
					panic("DynamicNamespaceScope: check function called multiple times")
				}
				called = true

				return h.checkNamespaceScope(r, namespace, user, scope)
			})

			if err != nil {
				var zero T
				return zero, fmt.Errorf("DynamicNamespaceScope: %w", err)
			}

			if !called {
				panic("DynamicNamespaceScope: check function not called")
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

// checkNamespaceScope evaluates if the given action is available for the given namespace.
//
// When the action is not available, an error of one of the following [api.ErrorCode]s is returned:
//
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.UnavailableInAnonymousMode]
// - [api.NamespaceNotFound]
// - [api.DatabaseError].
func (h *Handler) checkNamespaceScope(r *http.Request, namespace api.ValidNamespaceID, caller *api.Caller, scope scopes.NamespaceScope) error {
	if h.auth.AnonymousMode() {
		if err := scopes.EvaluateAnonymousModeNamespaceScope(namespace, scope); err != nil {
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

	if err := scopes.EvaluateNamespaceScope(namespace, role, caller, scope); err != nil {
		return fmt.Errorf("scope %q not fulfilled for namespace %q: %w", scope, namespace.String(), err)
	}
	return nil
}
