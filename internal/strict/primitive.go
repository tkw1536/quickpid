//spellchecker:words strict
package strict

//spellchecker:words bytes encoding json errors
import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
		return fmt.Errorf("Decoder.Token: %w", err)
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
		return fmt.Errorf("decoder.Token: %w", err)
	}

	boolean, ok := tok.(bool)
	if !ok {
		return errNotABoolean
	}
	*b = Bool(boolean)
	return nil
}
