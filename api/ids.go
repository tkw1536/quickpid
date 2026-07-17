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

// NamespaceID represents a valid namespace id.
//
// Use [NewNamespaceID] to create a new namespace id.
type NamespaceID struct {
	value string
}

// String returns the namespace id as a string.
func (ns NamespaceID) String() string {
	return ns.value
}

// regular expressions to validate various identifiers.
var (
	namespaceIDRE         = regexp.MustCompile(`^[a-z0-9_-]+$`)
	errInvalidNamespaceID = errors.New("invalid namespace id")
)

// NewNamespaceID creates a new NamespaceID.
func NewNamespaceID(value string) (*NamespaceID, error) {
	if !namespaceIDRE.MatchString(value) {
		return nil, errInvalidNamespaceID
	}
	return &NamespaceID{value: value}, nil
}
