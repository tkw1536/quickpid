package strict

import (
	"encoding/json"
	"fmt"
)

// Optional is a field that is can be used to unmarshal an optional fields inside a JSON object.
//
// It does not support marshalling, and is therefore only intended inside an implementation of the [json.Unmarshaler] interface.
type Optional[T any] struct {
	Value   T
	Present bool
}

// Set sets the value of this optional field.
func (opt *Optional[T]) Set(value T) {
	opt.Present = true
	opt.Value = value
}

// Clear clears the value of this optional field.
func (opt *Optional[T]) Clear() {
	var zero T
	opt.Present = false
	opt.Value = zero
}

func (r *Optional[T]) UnmarshalJSON(data []byte) error {
	var value T
	err := json.Unmarshal(data, &value)
	if err != nil {
		return fmt.Errorf("failed to unmarshal field: %w", err)
	}
	r.Set(value)
	return nil
}

// ToPointer returns a copy of this value as a pointer.
//
// The pointer is nil if and only if the value absent.
func (r Optional[T]) ToPointer() *T {
	if !r.Present {
		return nil
	}
	value := r.Value
	return &value
}
