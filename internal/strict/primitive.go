package strict

import (
	"bytes"
	"encoding/json"
	"errors"
)

var (
	errNotAString  = errors.New("can only unmarshal string literal")
	errNotABoolean = errors.New("can only unmarshal boolean literal")
)

// String rejects JSON null, and requires a string literal.
type String string

func (s *String) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return err
	}

	str, ok := tok.(string)
	if !ok {
		return errNotAString
	}
	*s = String(str)
	return nil
}

// String rejects JSON null, and requires a boolean literal.
type Bool bool

func (b *Bool) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return err
	}

	boolean, ok := tok.(bool)
	if !ok {
		return errNotABoolean
	}
	*b = Bool(boolean)
	return nil
}
