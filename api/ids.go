package api

//spellchecker:words errors regexp
import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
)

//spellchecker:words nolint recvcheck

// This file defines several seemingly useless structs,
// which at first glance look worse than a typedef string.
//
// However, defining them as structs has two benefits:
// 1. Creation (and validation with a regexp) has to be performed once, and only once.
// 2. Accidental casting from a string to an ID type cannot occur, and is guarded by the compiler.

// ValidNamespaceID represents a valid namespace id.
// The zero value is not valid.
//
// Use [NewNamespaceID] to create a new namespace id.
type ValidNamespaceID struct {
	valid bool
	value string
}

// String returns the namespace id as a string.
func (ns ValidNamespaceID) String() string {
	if !ns.valid {
		panic("invalid namespace id")
	}
	return ns.value
}

// regular expressions to validate various identifiers.
var (
	namespaceIDRE         = regexp.MustCompile(`^[a-z0-9_-]+$`)
	errInvalidNamespaceID = errors.New("invalid namespace id")
)

// NewNamespaceID creates a new NamespaceID.
func NewNamespaceID(value string) (ValidNamespaceID, error) {
	if !namespaceIDRE.MatchString(value) {
		return ValidNamespaceID{}, errInvalidNamespaceID
	}
	return ValidNamespaceID{valid: true, value: value}, nil
}

// ValidPID represents a valid pid.
//
// Use [NewPID] to create a new pid.
type ValidPID struct {
	valid bool
	value string
}

func (pid ValidPID) String() string {
	if !pid.valid {
		panic("invalid pid")
	}
	return pid.value
}

var (
	pidRE         = regexp.MustCompile(`^[a-z0-9_-]+$`)
	errInvalidPID = errors.New("invalid pid")
)

// NewPID creates a new PID.
func NewPID(value string) (ValidPID, error) {
	if !pidRE.MatchString(value) {
		return ValidPID{}, errInvalidPID
	}
	return ValidPID{valid: true, value: value}, nil
}

type ValidUsername struct {
	valid bool
	value string
}

func (username ValidUsername) String() string {
	if !username.valid {
		panic("invalid username")
	}
	return username.value
}

var (
	usernameRE         = regexp.MustCompile(`^[a-z0-9_-]+$`)
	errInvalidUsername = errors.New("invalid username")
)

// NewUsername creates a new Username.
func NewUsername(value string) (ValidUsername, error) {
	if !usernameRE.MatchString(value) {
		return ValidUsername{}, errInvalidUsername
	}
	return ValidUsername{valid: true, value: value}, nil
}

var (
	autocompleteQueryRE = regexp.MustCompile(`^[a-z0-9_-]+$`)
)

var errInvalidAutocompleteQuery = errors.New("invalid autocomplete query")

func NewAutocompleteQuery(value string) (ValidAutocompleteQuery, error) {
	if !autocompleteQueryRE.MatchString(value) {
		return ValidAutocompleteQuery{}, errInvalidAutocompleteQuery
	}
	return ValidAutocompleteQuery{valid: true, value: value}, nil
}

// ValidAutocompleteQuery represents a valid query for autocomplete.
//
// Use [NewAutocompleteQuery] to create a new autocomplete query.
type ValidAutocompleteQuery struct {
	valid bool
	value string
}

func (query ValidAutocompleteQuery) String() string {
	if !query.valid {
		panic("invalid autocomplete query")
	}
	return query.value
}

// ValidPassword represents a valid password.
type ValidPassword struct {
	valid bool
	value string
}

func (password ValidPassword) String() string {
	if !password.valid {
		panic("invalid password")
	}
	return password.value
}

var errInvalidPassword = errors.New("invalid password")

// NewPassword creates a new password.
func NewPassword(value string) (ValidPassword, error) {
	if value == "" {
		return ValidPassword{}, errInvalidPassword
	}
	return ValidPassword{valid: true, value: value}, nil
}

// ValidBaseURI represents a valid absolute base URI for a namespace mount.
// The zero value is not valid.
//
// Use [NewBaseURI] to create a new base URI.
type ValidBaseURI struct {
	valid bool
	value string
}

// String returns the base URI as a string.
func (u ValidBaseURI) String() string {
	if !u.valid {
		panic("invalid base uri")
	}
	return u.value
}

var (
	errNotAbsoluteURI   = errors.New("not an absolute URI")
	errFragmentNotEmpty = errors.New("fragment is not empty")
	errRawQueryNotEmpty = errors.New("raw query is not empty")
)

// NewBaseURI creates a new BaseURI.
// The value must be a valid absolute URI as accepted by [url.ParseRequestURI] with a non-empty scheme.
func NewBaseURI(value string) (ValidBaseURI, error) {
	parsed, err := url.Parse(value)
	if err != nil {
		return ValidBaseURI{}, fmt.Errorf("failed to parse as url: %w", err)
	}
	if !parsed.IsAbs() {
		return ValidBaseURI{}, errNotAbsoluteURI
	}
	if parsed.Fragment != "" {
		return ValidBaseURI{}, errFragmentNotEmpty
	}
	if parsed.RawQuery != "" {
		return ValidBaseURI{}, errRawQueryNotEmpty
	}
	return ValidBaseURI{valid: true, value: value}, nil
}

// ValidResourceURL represents a valid absolute URI for a resource target URL.
// The zero value is not valid.
//
// Use [NewResourceURL] to create a new resource URL.
//
//nolint:recvcheck // StringPtr method is intentionally used with a pointer.
type ValidResourceURL struct {
	valid bool
	value string
}

// String returns the resource URL as a string.
func (u ValidResourceURL) String() string {
	if !u.valid {
		panic("invalid resource url")
	}
	return u.value
}

// StringPtr returns the resource URL as a pointer to a string.
func (u *ValidResourceURL) StringPtr() *string {
	if u == nil {
		return nil
	}
	return new(u.value)
}

// NewResourceURL creates a new resource URL.
// The value must be a valid absolute URI with a non-empty scheme.
// Unlike [NewBaseURI], query strings and fragments are allowed.
func NewResourceURL(value string) (ValidResourceURL, error) {
	parsed, err := url.Parse(value)
	if err != nil {
		return ValidResourceURL{}, fmt.Errorf("failed to parse as url: %w", err)
	}
	if !parsed.IsAbs() {
		return ValidResourceURL{}, errNotAbsoluteURI
	}
	return ValidResourceURL{valid: true, value: value}, nil
}
