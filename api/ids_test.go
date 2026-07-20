package api_test

//spellchecker:words testing github quickpid
import (
	"testing"

	"github.com/tkw1536/quickpid/api"
)

func TestNewNamespaceID(t *testing.T) {
	t.Parallel()
	testIdentifierValidation(t, api.NewNamespaceID, "invalid namespace id")
}

func TestNewPID(t *testing.T) {
	t.Parallel()
	testIdentifierValidation(t, api.NewPID, "invalid pid")
}

func TestNewUsername(t *testing.T) {
	t.Parallel()
	testIdentifierValidation(t, api.NewUsername, "invalid username")
}

func testIdentifierValidation[T any](t *testing.T, validate func(string) (T, error), wantInvalidMsg string) {
	t.Helper()

	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:  "valid_alphanumeric",
			input: "abc123",
		},
		{
			name:  "valid_hyphen_underscore",
			input: "a-b_c",
		},
		{
			name:    "invalid_empty",
			input:   "",
			wantErr: wantInvalidMsg,
		},
		{
			name:    "invalid_uppercase",
			input:   "Alice",
			wantErr: wantInvalidMsg,
		},
		{
			name:    "invalid_special_characters",
			input:   "a.b",
			wantErr: wantInvalidMsg,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := validate(tt.input)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validate(%q) error = %v, want nil", tt.input, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("validate(%q) error = nil, want %q", tt.input, tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("validate(%q) error = %q, want %q", tt.input, err.Error(), tt.wantErr)
			}
		})
	}
}
