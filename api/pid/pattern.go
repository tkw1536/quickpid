package pid

import (
	"errors"
)

// Pattern represents a pattern for a PID.
//
// It should consist of only '*', '-', and '_' and must contain at least one '*'.
//
// When a PID is generated, the '*' characters are replaced with random characters from the character set.
// See [Format] for details.
type Pattern string

var (
	errInvalidCharacters = errors.New("pattern contains invalid characters")
	errMissingAsterisk   = errors.New("pattern must contain '*'")
)

// Valid checks if this is a valid pattern.
func (p Pattern) Valid() error {
	// check that the pattern consists only of '*', '-', and '_'.
	var hadAsterisk bool
	for _, c := range p {
		switch c {
		case '*':
			hadAsterisk = true
		case '-', '_':
		default:
			return errInvalidCharacters
		}
	}

	// must have at least one asterisk.
	if !hadAsterisk {
		return errMissingAsterisk
	}
	return nil
}
