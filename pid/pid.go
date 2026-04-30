// Package pid implements generation of a PID.
package pid

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/tkw1536/quickpid/internal/optional"
)

// Format describes the format of a PID.
type Format struct {
	Pattern    Pattern      `json:"pattern"`
	Characters CharacterSet `json:"characters"`
}

func (f *Format) UnmarshalJSON(data []byte) error {
	var internal struct {
		Pattern    optional.Optional[Pattern]      `json:"pattern"`
		Characters optional.Optional[CharacterSet] `json:"characters"`
	}
	if err := json.Unmarshal(data, &internal); err != nil {
		return fmt.Errorf("failed to unmarshal fields: %w", err)
	}
	if !internal.Pattern.Present {
		return fmt.Errorf("missing required field: pattern")
	}
	if !internal.Characters.Present {
		return fmt.Errorf("missing required field: characters")
	}
	f.Pattern = internal.Pattern.Value
	f.Characters = internal.Characters.Value
	return nil
}

var (
	errInvalidPattern      = errors.New("invalid pattern")
	errInvalidCharacterSet = errors.New("invalid character set")
)

// Validate checks if the format is valid, and returns an error if not.
func (format Format) Validate() error {
	if err := format.Characters.Validate(); err != nil {
		return fmt.Errorf("%w: %w", errInvalidCharacterSet, err)
	}

	if err := format.Pattern.Validate(); err != nil {
		return fmt.Errorf("%w: %w", errInvalidPattern, err)
	}
	return nil
}

// Generate generates a new PID according to format, using rand for randomness.
func (format Format) Generate(rand io.Reader) (string, error) {
	if err := format.Validate(); err != nil {
		return "", err
	}

	var writer strings.Builder
	writer.Grow(len(format.Pattern))

	for _, c := range format.Pattern {
		// copy non-asterisk characters directly
		if c != '*' {
			writer.WriteRune(c)
			continue
		}

		// otherwise pick a random character from the character set
		char, err := format.Characters.Pick(rand)
		if err != nil {
			return "", err
		}
		writer.WriteRune(char)
	}

	return writer.String(), nil
}
