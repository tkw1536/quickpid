// Package required checks for required fields in JSON objects
package required

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// missingFieldError indicates that a required field is missing from a JSON object.
type missingFieldError string

func (e missingFieldError) Error() string {
	return fmt.Sprintf("missing required field %q", string(e))
}

var (
	errInvalidJSON = errors.New("invalid JSON")
)

// Required checks that the given json object contains all fields.
func Required(data []byte, fields ...string) error {
	dec := json.NewDecoder(bytes.NewReader(data))

	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if tok != json.Delim('{') {
		return errInvalidJSON
	}

	needed := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		needed[f] = struct{}{}
	}

	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return err
		}

		// keys must be strings!
		key, ok := keyTok.(string)
		if !ok {
			return errInvalidJSON
		}

		// we encountered the key
		delete(needed, key)

		// skip the value for the value
		if err := skipValue(dec); err != nil {
			return err
		}
	}

	// should be at the closing token now!
	if _, err := dec.Token(); err != nil {
		return err
	}

	// trailing json not allowed
	if err := dec.Decode(new(struct{})); err != io.EOF {
		return errInvalidJSON
	}

	// we checked everything now, there shouldn't be any more fields.
	for _, f := range fields {
		if _, ok := needed[f]; ok {
			return missingFieldError(f)
		}
	}

	return nil
}

// skipValue skips the current value.
func skipValue(dec *json.Decoder) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}

	// a primitive value can be skipped immediately.
	delim, ok := tok.(json.Delim)
	if !ok {
		return nil
	}

	// now need to wait for the current object or array to be closed.
	// we can just count the opening and closing tokens.
	var (
		openToken  = delim
		closeToken json.Delim
	)
	switch openToken {
	case '{':
		closeToken = '}'
	case '[':
		closeToken = ']'
	default:
		return nil
	}

	depth := 1
	for depth > 0 {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		d, ok := tok.(json.Delim)
		if !ok {
			continue
		}

		switch d {
		case openToken:
			depth++
		case closeToken:
			depth--
		}
	}
	return nil
}
