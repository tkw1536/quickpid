//spellchecker:words lowlevel
package lowlevel

//spellchecker:words http github quickpid
import (
	"net/http"

	"github.com/tkw1536/quickpid/api"
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
		func(w http.ResponseWriter, r *http.Request, _ *api.ValidUserInfo) (T, error) {
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
	impl func(http.ResponseWriter, *http.Request, api.ValidUserInfo) (T, error),
	successCode func(T) int,
	allowedErrors []api.ErrorCode,
) http.HandlerFunc {
	return h.handle(
		requiredUser,
		func(w http.ResponseWriter, r *http.Request, user *api.ValidUserInfo) (T, error) {
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
	impl func(http.ResponseWriter, *http.Request, *api.ValidUserInfo) (T, error),
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
