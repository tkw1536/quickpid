package strict

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

var errMustBeStruct = errors.New("expected JSON object")

// MustBeStruct checks that the given data represents a JSON struct.
//
// If the data is not a valid JSON object, the behavior is undefined.
//
// It is intended to prevent un-marshalling of "null" into JSON structs.
func MustBeStruct(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	return mustBeStruct(dec)
}

func mustBeStruct(dec *json.Decoder) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if tok == json.Delim('{') {
		return nil
	}
	return fmt.Errorf("%w, got %v", errMustBeStruct, tok)
}

// UnmarshalStruct decodes a JSON object into a value of type T.
// T is expected to be a struct type.
//
// It rejects:
// - non-object inputs (e.g. null, arrays, strings)
// - unknown fields (like [json.Decoder.DisallowUnknownFields])
// - trailing non-whitespace after the JSON value
func UnmarshalStruct[T any](data []byte) (out T, err error) {

	// check that it's a struct
	bytesReader := bytes.NewReader(data)
	if err := mustBeStruct(json.NewDecoder(bytesReader)); err != nil {
		return out, err
	}

	// reset the reader and create a new decoder
	if _, err := bytesReader.Seek(0, io.SeekStart); err != nil {
		return out, err
	}
	dec := json.NewDecoder(bytesReader)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&out); err != nil {
		return out, err
	}

	// check that there isn't any trailing data
	if _, err := dec.Token(); err != io.EOF {
		return out, err
	}
	return out, nil
}
