// Package pid implements generation of a PID.
package pid

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/tkw1536/quickpid/internal/required"
)

// Format describes a format for a PID.
type Format struct {
	Pattern    Pattern      `json:"pattern"`
	Characters CharacterSet `json:"characters"`
}

func (f *Format) UnmarshalJSON(data []byte) error {
	if err := required.Required(data, "pattern", "characters"); err != nil {
		return err
	}

	type alias Format
	var out alias
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	*f = Format(out)
	return nil
}

var (
	errInvalidPattern      = errors.New("invalid pattern")
	errInvalidCharacterSet = errors.New("invalid character set")
)

// Valid checks if the format is valid, and returns an error if not.
func (format Format) Valid() error {
	if !format.Characters.Valid() {
		return errInvalidCharacterSet
	}

	if err := format.Pattern.Valid(); err != nil {
		return fmt.Errorf("%w: %w", errInvalidPattern, err)
	}
	return nil
}
func readFull(rand io.Reader, buf []byte) error {
	_, err := io.ReadFull(rand, buf)
	return err
}

// Generate generates a new PID according to format, using rand for randomness.

// GeneratePID generates a new PID according to format, using rand for randomness.
//
// It replaces each '*' in format.Pattern with a random character from format.Characters,
// and leaves '-' and '_' unchanged.
func (format Format) Generate(rand io.Reader) (string, error) {
	if err := format.Valid(); err != nil {
		return "", err
	}
	alphabet, _ := format.Characters.Alphabet()

	starCount := 0
	for i := 0; i < len(format.Pattern); i++ {
		if format.Pattern[i] == '*' {
			starCount++
		}
	}

	buf := make([]byte, starCount)
	if err := readFull(rand, buf); err != nil {
		return "", err
	}

	out := make([]byte, len(format.Pattern))
	n := byte(len(alphabet))
	j := 0
	for i := 0; i < len(format.Pattern); i++ {
		switch format.Pattern[i] {
		case '*':
			out[i] = alphabet[buf[j]%n]
			j++
		case '-', '_':
			out[i] = format.Pattern[i]
		}
	}
	return string(out), nil
}
