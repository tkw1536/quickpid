package api_test

//spellchecker:words errors testing github bicpid
import (
	"errors"
	"fmt"
	"testing"

	"github.com/tkw1536/bicpid/api"
)

var (
	errSomethingWentWrong = errors.New("something went wrong")
)

func ExampleWithErrorString() {
	// wrap an underlying error and add a specific error string.
	wrapped := api.WithErrorString(errSomethingWentWrong, api.DatabaseError)

	// they are identical in terms of Error()
	fmt.Println("errors are identical:", wrapped.Error() == errSomethingWentWrong.Error())
	fmt.Println("errors.Is for the underlying error:", errors.Is(wrapped, errSomethingWentWrong))

	// extract the error string
	str, ok := api.GetErrorString(wrapped)
	fmt.Println("got an error string back:", ok)
	fmt.Println("error string:", str)

	// Output: errors are identical: true
	// errors.Is for the underlying error: true
	// got an error string back: true
	// error string: database_error
}

func TestWithErrorString_GetErrorString(t *testing.T) {
	t.Parallel()

	annotated := api.WithErrorString(errSomethingWentWrong, api.DatabaseError)

	code, ok := api.GetErrorString(annotated)
	if !ok || code != api.DatabaseError {
		t.Fatalf("GetErrorString() = %q, %v, want %q, true", code, ok, api.DatabaseError)
	}
	if !errors.Is(annotated, errSomethingWentWrong) {
		t.Fatal("annotated error should unwrap to cause")
	}
}

func TestGetErrorString_notAnnotated(t *testing.T) {
	t.Parallel()

	_, ok := api.GetErrorString(errSomethingWentWrong)
	if ok {
		t.Fatal("GetErrorString() on plain error = true, want false")
	}
}

func TestGetErrorString_wrapped(t *testing.T) {
	t.Parallel()

	annotated := api.WithErrorString(errSomethingWentWrong, api.Forbidden)
	wrapped := fmt.Errorf("handler: %w", annotated)

	code, ok := api.GetErrorString(wrapped)
	if !ok || code != api.Forbidden {
		t.Fatalf("GetErrorString(wrapped) = %q, %v, want %q, true", code, ok, api.Forbidden)
	}
	if !errors.Is(wrapped, errSomethingWentWrong) {
		t.Fatal("wrapped error should unwrap to cause")
	}
}
