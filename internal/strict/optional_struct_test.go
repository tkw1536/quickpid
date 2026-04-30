package strict_test

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/tkw1536/quickpid/internal/strict"
)

var errRequiredMissing = errors.New("field 'required' is missing")

type MyStruct struct {
	Required *string  // may be null or string (but must be present) inside the json
	Optional **string // may be null, string, or absent inside the json
}

// PrintOptionalInfo prints the contents of the required and optional fields.
func (ms MyStruct) PrintInfo() {
	if ms.Required == nil {
		fmt.Print("Required == nil ")
	} else {
		fmt.Printf("Required == &(%q) ", *ms.Required)
	}

	switch {
	case ms.Optional == nil:
		fmt.Print("Optional == nil")
	case *ms.Optional == nil:
		fmt.Print("Optional == &(nil)")
	default:
		fmt.Printf("Optional == &(&(%q))", **ms.Optional)
	}

	fmt.Println()
}

func (ms *MyStruct) UnmarshalJSON(data []byte) error {
	// first marshal into an internal struct that has optional fields.
	var internal struct {
		Required strict.Optional[*string] `json:"required"`
		Optional strict.Optional[*string] `json:"optional"`
	}
	if err := json.Unmarshal(data, &internal); err != nil {
		return fmt.Errorf("failed to unmarshal fields: %w", err)
	}

	// check that the required field is present.
	if !internal.Required.Present {
		return errRequiredMissing
	}
	ms.Required = internal.Required.Value

	// set the optional field
	ms.Optional = internal.Optional.ToPointer()
	return nil
}

// Demonstrates how the optional field can be used to implement un-marshalling of both required and optional fields.
func ExampleOptional_customStruct() {
	// A missing required field is an error.
	var missingRequired MyStruct
	err := json.Unmarshal([]byte(`{}`), &missingRequired)
	fmt.Printf("Required field is missing returned error: %v\n", err)

	// The optional field can now distinguish between "absent" and "explicit null" (and also some value).

	var optionalFieldIsAbsent MyStruct
	_ = json.Unmarshal([]byte(`{"required":null}`), &optionalFieldIsAbsent)
	fmt.Print("Optional field is absent: ")
	optionalFieldIsAbsent.PrintInfo()

	var optionalFieldIsNull MyStruct
	_ = json.Unmarshal([]byte(`{"required":null,"optional":null}`), &optionalFieldIsNull)
	fmt.Print("Optional field is null: ")
	optionalFieldIsNull.PrintInfo()

	var optionalFieldIsPresent MyStruct
	_ = json.Unmarshal([]byte(`{"required":"hello","optional":"world"}`), &optionalFieldIsPresent)
	fmt.Print("Optional field is present: ")
	optionalFieldIsPresent.PrintInfo()

	// Output:
	// Required field is missing returned error: field 'required' is missing
	// Optional field is absent: Required == nil Optional == nil
	// Optional field is null: Required == nil Optional == &(nil)
	// Optional field is present: Required == &("hello") Optional == &(&("world"))
}
