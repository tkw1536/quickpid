//spellchecker:words service
package service

//spellchecker:words errors
import "errors"

var (
	errUnauthorized = errors.New("unauthorized")
	errForbidden    = errors.New("forbidden")

	errAnonymousModeUnavailable = errors.New("anonymous mode unavailable")

	errInsufficientEntropy = errors.New("insufficient entropy")
	errBadPID              = errors.New("bad pid generated")
	errBadNamespaceID      = errors.New("bad namespace id generated")
	errSpecInfoPrivate     = errors.New("info is private")

	errInvalidNamespaceID = errors.New("invalid namespace id")
	errInvalidPID         = errors.New("invalid pid")
	errInvalidUsername    = errors.New("invalid username")
)

// IsUnauthorized reports whether err indicates a missing or invalid API key.
func IsUnauthorized(err error) bool {
	return errors.Is(err, errUnauthorized)
}

// IsForbidden reports whether err indicates insufficient permissions.
func IsForbidden(err error) bool {
	return errors.Is(err, errForbidden)
}

// IsAnonymousModeUnavailable reports whether err indicates that the operation is unavailable in anonymous mode.
func IsAnonymousModeUnavailable(err error) bool {
	return errors.Is(err, errAnonymousModeUnavailable)
}
