// Package strict provides types that enable strict JSON unmarshaling.
package strict

func OptionalStringToPointer(value Optional[String]) *string {
	if !value.Present {
		return nil
	}
	return new(string(value.Value))
}

func OptionalBoolToPointer(value Optional[Bool]) *bool {
	if !value.Present {
		return nil
	}
	return new(bool(value.Value))
}
