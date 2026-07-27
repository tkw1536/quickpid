//spellchecker:words lowlevel
package lowlevel

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

// HandlePublic returns a handler for a public endpoint, that is one which does not require authentication.
func HandlePublic[T any](
	h *Handler,
	impl func(http.ResponseWriter, *http.Request) (T, error),
	successCode func(T) int,
	allowedErrors []api.ErrorString,
) http.HandlerFunc {
	return handle(
		h,
		noAuthentication,
		func(w http.ResponseWriter, r *http.Request, _ *api.ValidUserInfo) (T, error) {
			return impl(w, r)
		},
		successCode,
		allowedErrors,
	)
}

// HandleRestricted handles a request for a restricted endpoint, that is one which requires a signed-in user.
//
// When the server is in anonymous mode, returns [api.UnavailableInAnonymousMode].
// When in authenticated mode, can return [api.Unauthorized] and [api.DatabaseError].
func HandleRestricted[T any](
	h *Handler,
	impl func(http.ResponseWriter, *http.Request, api.ValidUserInfo) (T, error),
	successCode func(T) int,
	allowedErrors []api.ErrorString,
) http.HandlerFunc {
	return handle(
		h,
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

// HandleOpen handles a request for an open endpoint, optionally requiring a signed-in user.
//
// When in anonymous mode, returns no additional errors.
// When in authenticated mode, can return [api.Unauthorized] and [api.DatabaseError].
func HandleOpen[T any](
	h *Handler,
	impl func(http.ResponseWriter, *http.Request, *api.ValidUserInfo) (T, error),
	successCode func(T) int,
	allowedErrors []api.ErrorString,
) http.HandlerFunc {
	return handle(
		h,
		optionalUser,
		impl,
		successCode,
		allowedErrors,
	)
}
