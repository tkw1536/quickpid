//spellchecker:words auth
package auth

//spellchecker:words bytes encoding json errors http github htpasswd pkglib lazy
import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/tg123/go-htpasswd"
	"go.tkw01536.de/pkglib/lazy"
)

// parsers lists supported htpasswd hash formats for [BasicAuthentication].
// Plaintext hashes are intentionally excluded.
var parsers = []htpasswd.PasswdParser{
	htpasswd.AcceptMd5,
	htpasswd.AcceptSha,
	htpasswd.AcceptBcrypt,
}

// BasicAuthentication performs timing-safe HTTP basic authentication.
//
// Once used, a [BasicAuthentication] struct MUST NOT be modified.
type BasicAuthentication struct {

	// Users holds a list of username:hash pairs.
	// Hashes correspond to the htpasswd format, and can be generated using something like
	// `htpasswd -n username` on the command line.
	Users []string

	// cached htpasswd file
	file lazy.Lazy[*htpasswd.File]
}

// MarshalJSON encodes users as a JSON array of username:hash lines.
func (ba *BasicAuthentication) MarshalJSON() ([]byte, error) {
	return json.Marshal(ba.Users)
}

// UnmarshalJSON decodes users from a JSON array of username:hash lines.
func (ba *BasicAuthentication) UnmarshalJSON(data []byte) error {
	var users []string
	if err := json.Unmarshal(data, &users); err != nil {
		return err
	}
	ba.Users = users
	ba.file.Set(ba.processUsers())
	return nil
}

// processUsers parses the selected users into a htpasswd file.
func (ba *BasicAuthentication) processUsers() *htpasswd.File {
	var content bytes.Buffer
	for _, user := range ba.Users {
		content.WriteString(string(user))
		content.WriteByte('\n')
	}

	file, err := htpasswd.NewFromReader(&content, parsers, nil)
	if err != nil {
		// If there is an error, explicitly set the file to nil.
		// The underlying function might already do this, but we do it explicitly to be safe.
		file = nil
	}
	return file

}

var errUnauthorized = errors.New("unauthorized")

// Authenticate performs basic authentication safely and returns the username.
// If there is no authorized user, returns [errUnauthorized].
func (ba *BasicAuthentication) Authenticate(r *http.Request) (string, error) {
	username, password, ok := r.BasicAuth()
	if !ok {
		return "", errUnauthorized
	}

	file := ba.file.Get(ba.processUsers)
	if file == nil { // couldn't parse for some reason
		return "", errUnauthorized
	}

	if !file.Match(username, password) {
		return "", errUnauthorized
	}

	return username, nil
}
