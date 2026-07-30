//spellchecker:words strict
package strict

//spellchecker:words bytes encoding json errors
import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var (
	errNotAString  = errors.New("can only unmarshal string literal")
	errNotABoolean = errors.New("can only unmarshal boolean literal")
	errNotAnArray  = errors.New("can only unmarshal JSON array")
	errNotRFC3339  = errors.New("can only unmarshal RFC3339 time string")
)

// String rejects JSON null, and requires a string literal.
type String string

func (s *String) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return fmt.Errorf("failed to read first token: %w", err)
	}

	str, ok := tok.(string)
	if !ok {
		return errNotAString
	}
	*s = String(str)
	return nil
}

// Bool rejects JSON null, and requires a boolean literal.
type Bool bool

func (b *Bool) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return fmt.Errorf("failed to read first token: %w", err)
	}

	boolean, ok := tok.(bool)
	if !ok {
		return errNotABoolean
	}
	*b = Bool(boolean)
	return nil
}

// Time rejects JSON null, requires a string literal, and parses RFC3339.
type Time time.Time

func (t *Time) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return fmt.Errorf("failed to read first token: %w", err)
	}

	str, ok := tok.(string)
	if !ok {
		return errNotAString
	}
	parsed, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return fmt.Errorf("%w: %w", errNotRFC3339, err)
	}
	*t = Time(parsed)
	return nil
}

// Time returns the underlying [time.Time] value.
func (t *Time) Time() time.Time {
	return time.Time(*t)
}

// StringSlice rejects JSON null and non-arrays, and requires each element to be a string literal.
type StringSlice []String

func (s *StringSlice) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return fmt.Errorf("failed to read first token: %w", err)
	}
	if tok != json.Delim('[') {
		return errNotAnArray
	}

	var out []String
	for dec.More() {
		var elem String
		if err := dec.Decode(&elem); err != nil {
			return fmt.Errorf("failed to decode json: %w", err)
		}
		out = append(out, elem)
	}
	tok, err = dec.Token()
	if err != nil {
		return fmt.Errorf("failed to read last token: %w", err)
	}
	if tok != json.Delim(']') {
		return errNotAnArray
	}
	if out == nil {
		out = []String{}
	}
	*s = out
	return nil
}

// Strings converts this slice to a plain []string.
func (s *StringSlice) Strings() []string {
	out := make([]string, len(*s))
	for i, v := range *s {
		out[i] = string(v)
	}
	return out
}
