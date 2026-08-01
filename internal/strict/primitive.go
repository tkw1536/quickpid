// Package strict provides types that enable strict JSON unmarshaling.
//
//spellchecker:words strict
package strict

//spellchecker:words bytes encoding json errors time
import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
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
	dec := jsontext.NewDecoder(bytes.NewReader(data))
	tok, err := dec.ReadToken()
	if err != nil {
		return fmt.Errorf("failed to read first token: %w", err)
	}

	if tok.Kind() != jsontext.KindString {
		return errNotAString
	}

	*s = String(tok.String())
	return nil
}

// Bool rejects JSON null, and requires a boolean literal.
type Bool bool

func (b *Bool) UnmarshalJSON(data []byte) error {
	dec := jsontext.NewDecoder(bytes.NewReader(data))
	tok, err := dec.ReadToken()
	if err != nil {
		return fmt.Errorf("failed to read first token: %w", err)
	}

	var value bool
	switch tok.Kind() {
	case jsontext.KindTrue:
		value = true
	case jsontext.KindFalse:
		value = false
	case jsontext.KindInvalid, jsontext.KindNull, jsontext.KindString, jsontext.KindNumber, jsontext.KindBeginObject, jsontext.KindEndObject, jsontext.KindBeginArray, jsontext.KindEndArray:
		return errNotABoolean
	default:
		panic("never reached")
	}

	*b = Bool(value)
	return nil
}

// Time rejects JSON null, requires a string literal, and parses RFC3339.
type Time time.Time

func (t *Time) UnmarshalJSON(data []byte) error {
	dec := jsontext.NewDecoder(bytes.NewReader(data))
	tok, err := dec.ReadToken()
	if err != nil {
		return fmt.Errorf("failed to read first token: %w", err)
	}

	if tok.Kind() != jsontext.KindString {
		return errNotAString
	}
	parsed, err := time.Parse(time.RFC3339, tok.String())
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
	dec := jsontext.NewDecoder(bytes.NewReader(data))
	tok, err := dec.ReadToken()
	if err != nil {
		return fmt.Errorf("failed to read first token: %w", err)
	}
	if tok.Kind() != jsontext.KindBeginArray {
		return errNotAnArray
	}

	var out []String
	for {
		kind := dec.PeekKind()
		if kind == jsontext.KindEndArray || kind == jsontext.KindInvalid {
			break
		}
		var elem String
		if err := json.UnmarshalDecode(dec, &elem); err != nil {
			return fmt.Errorf("failed to decode json: %w", err)
		}
		out = append(out, elem)
	}
	tok, err = dec.ReadToken()
	if err != nil {
		return fmt.Errorf("failed to read last token: %w", err)
	}
	if tok.Kind() != jsontext.KindEndArray {
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
