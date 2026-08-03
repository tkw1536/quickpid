//spellchecker:words credentials
package credentials

//spellchecker:words http strings
import (
	"net/http"
	"strings"
)

// Credentials represents parsed credentials from an Authorization header.
//
// Credentials are of a specific [Kind], which can be retrieved using the [Credentials.Kind] method.
// - [KindEmpty] represents a header that does not contain any credentials.
// - [KindBasic] represents a header that contains "HTTP Basic" credentials which can be retrieved using [Credentials.BasicAuth].
// - [KindBearer] represents a header that contains a "Bearer" token which can be retrieved using [Credentials.BearerToken].
// - [KindInvalid] represents any other case.
//
// Credentials are intended to be created using [Parse].
// The zero value of Credentials represents empty credentials.
type Credentials struct {
	invalid bool

	basic                        bool
	basicUsername, basicPassword string

	bearer      bool
	bearerValue string
}

// Kind represents the kind of credentials contained in a [Credentials] object.
type Kind int

const (
	KindInvalid Kind = iota
	KindEmpty
	KindBasic
	KindBearer
)

const (
	basicPrefix  = "Basic "
	bearerPrefix = "Bearer "
)

// Parse creates a new [Credentials] by parsing them from request headers in h.
func Parse(h http.Header) Credentials {
	var creds Credentials
	for _, auth := range h.Values("Authorization") {
		switch {
		case hasPrefixAsciiFold(auth, bearerPrefix):
			if creds.basic || creds.bearer {
				return Credentials{invalid: true}
			}
			creds.bearer = true
			creds.bearerValue = strings.TrimSpace(trimPrefixAsciiFold(auth, bearerPrefix))
		case hasPrefixAsciiFold(auth, basicPrefix):
			if creds.basic || creds.bearer {
				// saw previous credentials
				return Credentials{invalid: true}
			}

			// HACK HACK HACK: To parse the basic auth from the header, we need to create a request.
			// because we cannot directly access any parseBasicAuth function.
			//
			// TODO: Consider writing the method ourselves.
			r := http.Request{Header: h}
			creds.basicUsername, creds.basicPassword, creds.basic = r.BasicAuth()
			if !creds.basic {
				return Credentials{invalid: true}
			}

		default:
			return Credentials{invalid: true}
		}
	}
	return creds
}

// Kind determines the kind of credentials this object represents.
func (h *Credentials) Kind() Kind {
	switch {
	case h.invalid, h.basic && h.bearer:
		return KindInvalid
	case !h.basic && !h.bearer:
		return KindEmpty
	case h.basic:
		return KindBasic
	case h.bearer:
		return KindBearer
	}
	panic("never reached")
}

// BasicAuth returns the contained basic username and password.
// If the Header does not contain a basic username and password, the function panics.
func (h *Credentials) BasicAuth() (username, password string) {
	if !h.basic || h.invalid {
		panic("Kind() !== KindBasic")
	}
	return h.basicUsername, h.basicPassword
}

// BearerToken returns the contained bearer token.
// If the Header does not contain a bearer token, the function panics.
func (h *Credentials) BearerToken() (token string) {
	if !h.bearer || h.invalid {
		panic("Kind() !== KindBearer")
	}
	return h.bearerValue
}
