//spellchecker:words service
package service

//spellchecker:words errors
import (
	"errors"
)

var (
	errUnauthorized  = errors.New("unauthorized")
	errForbidden     = errors.New("forbidden")
	errForbiddenSelf = errors.New("forbidden to clear your own password")

	errInsufficientEntropy = errors.New("insufficient entropy")
	errBadPID              = errors.New("bad pid generated")
	errBadNamespaceID      = errors.New("bad namespace id generated")
	errSpecInfoPrivate     = errors.New("info is private")

	errExpiresAtInPast = errors.New("expires_at is in the past")
)
