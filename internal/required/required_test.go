package required

import (
	"errors"
	"testing"
)

func TestRequired_OK(t *testing.T) {
	data := []byte(`{"a":1,"c":{},"b":null}`)
	if err := Required(data, "a", "b"); err != nil {
		t.Fatalf("Required returned error: %v", err)
	}
}

func TestRequired_MissingField(t *testing.T) {
	data := []byte(`{"a":1}`)
	err := Required(data, "a", "b")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.As(err, new(missingFieldError)) {
		t.Fatalf("expected missingFieldError, got %T: %v", err, err)
	}
	if got, want := err.Error(), `missing required field "b"`; got != want {
		t.Fatalf("error message mismatch: got %q want %q", got, want)
	}
}

func TestRequired_InvalidJSON(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{name: "malformed", data: []byte(`{`)},
		{name: "non_object", data: []byte(`[]`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Required(tt.data, "a"); err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}
