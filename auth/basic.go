//spellchecker:words auth
package auth

//spellchecker:words slog http
import (
	"log/slog"
	"net/http"
)

const basicAuthRealm = "quickpid"

// BasicAuthorization implements [Authorization] using http basic authentication.
//
// Once used, a [BasicAuthorization] struct MUST NOT be modified.
type BasicAuthorization struct {
	Authentication BasicAuthentication `json:"users"`

	// Permissions holds levels for each namespace and level.
	// The outer key is the namespace, the inner key is the username, the final value is the level.
	// Users not explicitly listed in the map are considered to have [LevelNone].
	Permissions map[string]map[string]Level `json:"permissions"`

	// Superusers can create namespaces and automatically get [LevelEditor] for all namespaces.
	Superusers []string `json:"superusers"`
}

// isSuperuser checks if the given user is a superuser.
func (ba *BasicAuthorization) isSuperuser(username string) bool {
	count := 0
	for _, superuser := range ba.Superusers {
		if superuser == username {
			count++
		}
	}
	return count > 0
}

func (ba *BasicAuthorization) Level(l *slog.Logger, r *http.Request, namespace string) (Level, error) {
	username, err := ba.Authentication.Authenticate(r)
	if err != nil {
		return LevelNone, errUnauthorized
	}

	if ba.isSuperuser(username) {
		return LevelEditor, nil
	}

	level, ok := ba.Permissions[namespace][username]
	if !ok {
		level = LevelNone
	}
	return level, nil
}

func (ba *BasicAuthorization) CanCreate(l *slog.Logger, r *http.Request) (bool, error) {
	username, err := ba.Authentication.Authenticate(r)
	if err != nil {
		return false, errUnauthorized
	}

	return ba.isSuperuser(username), nil
}

func (ba *BasicAuthorization) OnCreateNamespace(l *slog.Logger, r *http.Request, namespace string) error {
	return nil
}

func (ba *BasicAuthorization) OnUnauthorized(l *slog.Logger, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", `Basic realm="`+basicAuthRealm+`"`)
	http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
}
