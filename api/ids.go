package api

//spellchecker:words errors regexp
import (
	"errors"
	"regexp"
)

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
