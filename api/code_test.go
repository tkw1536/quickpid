package api_test

//spellchecker:words errors testing github quickpid
import (
	"errors"
	"fmt"
	"testing"

	"github.com/tkw1536/quickpid/api"
)

var (
	errSomethingWentWrong = errors.New("something went wrong")
)

func ExampleWithErrorCode() {
	// wrap an underlying error and add a specific error string.
	wrapped := api.WithErrorCode(errSomethingWentWrong, api.DatabaseError)

	// they are identical in terms of Error()
	fmt.Println("errors are identical:", wrapped.Error() == errSomethingWentWrong.Error())
	fmt.Println("errors.Is for the underlying error:", errors.Is(wrapped, errSomethingWentWrong))

	// extract the error string
	code, ok := api.GetErrorCode(wrapped)
	fmt.Println("got an error code back:", ok)
	fmt.Println("error code:", code.String())

	// Output: errors are identical: true
	// errors.Is for the underlying error: true
	// got an error code back: true
	// error code: databaseError
}

func TestWithErrorCode_GetErrorCode(t *testing.T) {
	t.Parallel()

	annotated := api.WithErrorCode(errSomethingWentWrong, api.DatabaseError)

	code, ok := api.GetErrorCode(annotated)
	if !ok || code != api.DatabaseError {
		t.Fatalf("GetErrorCode() = %q, %v, want %q, true", code, ok, api.DatabaseError)
	}
	if !errors.Is(annotated, errSomethingWentWrong) {
		t.Fatal("annotated error should unwrap to cause")
	}
}

func TestGetErrorCode_notAnnotated(t *testing.T) {
	t.Parallel()

	_, ok := api.GetErrorCode(errSomethingWentWrong)
	if ok {
		t.Fatal("GetErrorCode() on plain error = true, want false")
	}
}

func TestGetErrorCode_wrapped(t *testing.T) {
	t.Parallel()

	annotated := api.WithErrorCode(errSomethingWentWrong, api.Forbidden)
	wrapped := fmt.Errorf("handler: %w", annotated)

	code, ok := api.GetErrorCode(wrapped)
	if !ok || code != api.Forbidden {
		t.Fatalf("GetErrorCode(wrapped) = %q, %v, want %q, true", code, ok, api.Forbidden)
	}
	if !errors.Is(wrapped, errSomethingWentWrong) {
		t.Fatal("wrapped error should unwrap to cause")
	}
}

func TestErrorCode_HTTPCode(t *testing.T) {
	t.Parallel()

	if got := api.InvalidPassword.HTTPCode(); got != 400 {
		t.Fatalf("api.InvalidPassword.HTTPCode() = %d, want 400", got)
	}
}
