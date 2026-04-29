package pid

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

// A CharacterSet to be used in pid generation.
// Must be one of the pre-defined character sets.
type CharacterSet string

// Pre-defined character sets.
const (
	Full     CharacterSet = "full"
	Readable CharacterSet = "readable"
	Hex      CharacterSet = "hex"
	Decimal  CharacterSet = "decimal"
)

var alphabets = map[CharacterSet]string{
	Full:     "0123456789abcdefghijklmnopqrstuvwxyz",
	Readable: "0123456789abcdefghjkmnpqrstvwxyz",
	Hex:      "0123456789abcdef",
	Decimal:  "0123456789",
}

var (
	errUnknownCharacterSet = errors.New("unknown character set")
	errEmptyCharacterSet   = errors.New("empty character set")
	errFailedToReadRandom  = errors.New("failed to read random")
	errAllAttemptsFailed   = errors.New("repeated attempts to pick character fell outside of range")
)

// Validate checks if this is a valid character set.
func (chars CharacterSet) Validate() error {
	_, ok := alphabets[chars]
	if !ok {
		return errUnknownCharacterSet
	}
	return nil
}

// Alphabet returns the alphabet to be used for the given character set.
func (chars CharacterSet) Alphabet() (string, bool) {
	alphabet, ok := alphabets[chars]
	return alphabet, ok
}

// maxPickRetryAttempts is the maximum number of attempts to pick a character before giving up.
// If you update this constant, remember to update the tests.
const maxPickRetryAttempts = 100

// Pick picks a random rune from the character set, using rand for randomness.
func (chars CharacterSet) Pick(rand io.Reader) (rune, error) {
	alphabet, ok := chars.Alphabet()
	if !ok {
		return 0, errUnknownCharacterSet
	}

	runes := []rune(alphabet)
	n := uint64(len(runes))
	if n == 0 {
		return 0, errEmptyCharacterSet
	}

	// The algorithm first selects a random unit64 (8 bytes).
	// We then want to use this to uniformly select a rune from the alphabet.
	//
	// Modding out the length of the alphabet is only uniform if the maximum uint64 is a multiple of the length of the alphabet.
	// We enforce this by only accepting random ints in the range [0, T] where T is the maximum multiple of the length of the alphabet to fit into uint64.
	//
	// For practical purposes, we also limit the number of retries, and bail out if above that.
	maxMultiple := math.MaxUint64 - (math.MaxUint64 % n)

	var buffer [8]byte
	for range maxPickRetryAttempts {
		_, err := rand.Read(buffer[:])
		if err != nil {
			return 0, fmt.Errorf("%w: %w", errFailedToReadRandom, err)
		}

		choice := binary.BigEndian.Uint64(buffer[:])
		if choice >= maxMultiple {
			continue
		}

		return runes[choice%n], nil
	}

	return 0, errAllAttemptsFailed
}
