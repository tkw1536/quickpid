package strict

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

var errMustBeStruct = errors.New("expected JSON object")

// MustBeStruct checks that the given data represents a JSON struct.
//
// If the data is not a valid JSON object, the behavior is undefined.
//
// It is intended to prevent un-marshalling of "null" into JSON structs.
func MustBeStruct(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if tok == json.Delim('{') {
		return nil
	}

	return fmt.Errorf("%w, got %v", errMustBeStruct, tok)
}
