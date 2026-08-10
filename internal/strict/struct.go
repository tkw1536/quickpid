//spellchecker:words strict
package strict

//spellchecker:words bytes encoding json jsontext errors
import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
)

var errMustBeStruct = errors.New("expected JSON object")

// MustBeStruct checks that the given data represents a JSON struct.
//
// If the data is not a valid JSON object, it returns an error.
//
// It is intended to prevent un-marshalling of "null" into JSON structs.
func MustBeStruct(data []byte) error {
	return mustBeStruct(bytes.NewReader(data))
}

func mustBeStruct(r io.Reader) error {
	tok, err := jsontext.NewDecoder(r).ReadToken()
	if err != nil {
		return fmt.Errorf("failed to read first token: %w", err)
	}
	if tok.Kind() != jsontext.KindBeginObject {
		return fmt.Errorf("%w, got %v", errMustBeStruct, tok)
	}
	return nil
}

// UnmarshalStruct decodes a JSON object into a value of type T.
// T is expected to be a struct type.
//
// It rejects:
// - non-object inputs (e.g. null, arrays, strings)
// - unknown fields (like [json.RejectUnknownMembers])
// - trailing non-whitespace after the JSON value.
func UnmarshalStruct[T any](data []byte) (out T, err error) {
	// check that it's a struct
	bytesReader := bytes.NewReader(data)
	if err := mustBeStruct(bytesReader); err != nil {
		return out, fmt.Errorf("failed to check that data is a struct: %w", err)
	}

	// reset the reader and create a new decoder
	if _, err := bytesReader.Seek(0, io.SeekStart); err != nil {
		return out, fmt.Errorf("failed to reset reader: %w", err)
	}
	dec := jsontext.NewDecoder(bytesReader, json.RejectUnknownMembers(true))

	if err := json.UnmarshalDecode(dec, &out); err != nil {
		return out, fmt.Errorf("failed to decode json: %w", err)
	}

	// check that there isn't any trailing data
	if _, err := dec.ReadToken(); !errors.Is(err, io.EOF) {
		return out, fmt.Errorf("json has trailing data: %w", err)
	}
	return out, nil
}
