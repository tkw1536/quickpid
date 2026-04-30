package optional_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/tkw1536/quickpid/internal/optional"
)

// Demonstrates how to use the optional field can help differentiate between missing and null fields.
func ExampleOptional() {
	type Payload struct {
		S optional.Optional[*string]
	}

	var missing Payload
	_ = json.Unmarshal([]byte(`{}`), &missing)
	fmt.Printf("missing: Present=%#v Value=%#v\n", missing.S.Present, missing.S.Value)

	var explicitNull Payload
	_ = json.Unmarshal([]byte(`{"S":null}`), &explicitNull)
	fmt.Printf("null:    Present=%#v Value=%#v\n", explicitNull.S.Present, explicitNull.S.Value)

	var someValue Payload
	_ = json.Unmarshal([]byte(`{"S":"hello"}`), &someValue)
	fmt.Printf("someValue: Present=%#v Value=%#v\n", someValue.S.Present, *someValue.S.Value)

	// Output:
	// missing: Present=false Value=(*string)(nil)
	// null:    Present=true Value=(*string)(nil)
	// someValue: Present=true Value="hello"
}

func TestOptional_SetClearToPointer(t *testing.T) {
	t.Run("ToPointer_nilWhenAbsent", func(t *testing.T) {
		var opt optional.Optional[string]
		if got := opt.ToPointer(); got != nil {
			t.Fatalf("ToPointer() = %v, want nil", *got)
		}
	})

	t.Run("ToPointer_pointsToCopy", func(t *testing.T) {
		var opt optional.Optional[string]
		opt.Set("a")

		p := opt.ToPointer()
		if p == nil {
			t.Fatalf("ToPointer() = nil, want non-nil")
		}
		if *p != "a" {
			t.Fatalf("ToPointer() deref = %q, want %q", *p, "a")
		}

		*p = "b"
		if opt.Value != "a" {
			t.Fatalf("mutating ToPointer() result changed opt.Value to %q, want %q", opt.Value, "a")
		}
	})

	t.Run("Clear_resetsToZeroAndAbsent", func(t *testing.T) {
		var opt optional.Optional[int]
		opt.Set(123)
		opt.Clear()

		if opt.Present {
			t.Fatalf("Present = true, want false after Clear()")
		}
		if opt.Value != 0 {
			t.Fatalf("Value = %v, want zero after Clear()", opt.Value)
		}
	})
}

func TestOptional_JSON_UnmarshalDirect(t *testing.T) {
	t.Run("setsPresentOnSuccess", func(t *testing.T) {
		var opt optional.Optional[int]
		if err := json.Unmarshal([]byte(`123`), &opt); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !opt.Present {
			t.Fatalf("Present = false, want true")
		}
		if opt.Value != 123 {
			t.Fatalf("Value = %v, want %v", opt.Value, 123)
		}
	})

	t.Run("propagatesErrorWithContext", func(t *testing.T) {
		var opt optional.Optional[int]
		err := json.Unmarshal([]byte(`"not-an-int"`), &opt)
		if err == nil {
			t.Fatalf("unmarshal: got nil error, want error")
		}
		if got := err.Error(); !strings.HasPrefix(got, "failed to unmarshal field:") {
			t.Fatalf("error = %q, want prefix %q", got, "failed to unmarshal field:")
		}
	})
}
