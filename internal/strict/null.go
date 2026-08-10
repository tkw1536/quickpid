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

	"github.com/tkw1536/quickpid/internal/seekzero"
)

var (
	ErrJsonNull       = errors.New("json is null")
	ErrFailedToDecode = errors.New("failed to decode json")
	ErrTrailingData   = errors.New("json has trailing data")
)

// UnmarshalStrict strictly decodes valid JSON into a value of type T.
//
// It rejects:
// - null inputs, returning an error wrapping [ErrJsonNull]
// - unknown fields (like [json.RejectUnknownMembers]) returning an error wrapping [ErrFailedToDecode]
// - trailing non-whitespace after the JSON value returning an error wrapping [ErrTrailingData]
func UnmarshalStrict[T any](data []byte) (T, error) {
	var out T
	err := UnmarshalStrictTo(bytes.NewReader(data), &out)
	return out, err
}

// UnmarshalStrictTo is like [UnmarshalStrict].
// It accepts an [io.Reader] instead of a byte slice.
// It unmarshals the JSON into the given value.
func UnmarshalStrictTo(reader io.Reader, out any) error {
	seekable := seekzero.MakeOnceSeekable(reader)
	if err := mustNotBeNull(seekable); err != nil {
		return fmt.Errorf("%w: %w", ErrJsonNull, err)
	}

	if _, err := seekable.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToDecode, err)
	}

	dec := jsontext.NewDecoder(seekable, json.RejectUnknownMembers(true))
	if err := json.UnmarshalDecode(dec, out); err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToDecode, err)
	}

	// check that there isn't any trailing data
	if _, err := dec.ReadToken(); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: %w", ErrTrailingData, err)
	}
	return nil
}

// mustNotBeNull checks that the first json token in r is not null.
func mustNotBeNull(r io.Reader) error {
	tok, err := jsontext.NewDecoder(r).ReadToken()
	if err != nil {
		return fmt.Errorf("failed to read first token: %w", err)
	}
	if tok.Kind() == jsontext.KindNull {
		return ErrJsonNull
	}
	return nil
}
