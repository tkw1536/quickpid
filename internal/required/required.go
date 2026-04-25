// Package required checks for required fields in JSON objects
package required

import (
	"encoding/json"
	"fmt"
)

// missingFieldError indicates that a required field is missing from a JSON object.
type missingFieldError string

func (e missingFieldError) Error() string {
	return fmt.Sprintf("missing required field %q", string(e))
}

// Required checks that the given json object contains all fields.
func Required(data []byte, fields ...string) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for _, f := range fields {
		if _, ok := raw[f]; !ok {
			return missingFieldError(f)
		}
	}
	return nil
}
