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
//
// Use [NewNamespaceID] to create a new namespace id.
type ValidNamespaceID struct {
	value string
}

// String returns the namespace id as a string.
func (ns ValidNamespaceID) String() string {
	return ns.value
}

// regular expressions to validate various identifiers.
var (
	namespaceIDRE         = regexp.MustCompile(`^[a-z0-9_-]+$`)
	errInvalidNamespaceID = errors.New("invalid namespace id")
)

// NewNamespaceID creates a new NamespaceID.
func NewNamespaceID(value string) (*ValidNamespaceID, error) {
	if !namespaceIDRE.MatchString(value) {
		return nil, errInvalidNamespaceID
	}
	return &ValidNamespaceID{value: value}, nil
}

// ValidPID represents a valid pid.
//
// Use [NewPID] to create a new pid.
type ValidPID struct {
	value string
}

func (pid ValidPID) String() string {
	return pid.value
}

var (
	pidRE         = regexp.MustCompile(`^[a-z0-9_-]+$`)
	errInvalidPID = errors.New("invalid pid")
)

// NewPID creates a new PID.
func NewPID(value string) (*ValidPID, error) {
	if !pidRE.MatchString(value) {
		return nil, errInvalidPID
	}
	return &ValidPID{value: value}, nil
}

type ValidUsername struct {
	value string
}

func (username ValidUsername) String() string {
	return username.value
}

var (
	usernameRE         = regexp.MustCompile(`^[a-z0-9_-]+$`)
	errInvalidUsername = errors.New("invalid username")
)

// NewUsername creates a new Username.
func NewUsername(value string) (*ValidUsername, error) {
	if !usernameRE.MatchString(value) {
		return nil, errInvalidUsername
	}
	return &ValidUsername{value: value}, nil
}
