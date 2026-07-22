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

func TestNewPassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:  "valid_simple",
			input: "secret",
		},
		{
			name:  "valid_with_spaces",
			input: "secret phrase",
		},
		{
			name:    "invalid_empty",
			input:   "",
			wantErr: "invalid password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			password, err := api.NewPassword(tt.input)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("NewPassword(%q) error = %v, want nil", tt.input, err)
				}
				if password.String() != tt.input {
					t.Fatalf("password.String() = %q, want %q", password.String(), tt.input)
				}
				return
			}
			if err == nil {
				t.Fatalf("NewPassword(%q) error = nil, want %q", tt.input, tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("NewPassword(%q) error = %q, want %q", tt.input, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestNewBaseURI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:  "valid_https",
			input: "https://example.com/prefix/",
		},
		{
			name:  "valid_http_with_port",
			input: "http://localhost:8080/api",
		},
		{
			name:    "invalid_empty",
			input:   "",
			wantErr: "invalid base uri: not an absolute URI",
		},
		{
			name:    "invalid_relative",
			input:   "/relative/path",
			wantErr: "invalid base uri: not an absolute URI",
		},
		{
			name:    "invalid_no_scheme",
			input:   "example.com/foo",
			wantErr: "invalid base uri: not an absolute URI",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uri, err := api.NewBaseURI(tt.input)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("NewBaseURI(%q) error = %v, want nil", tt.input, err)
				}
				if uri.String() != tt.input {
					t.Fatalf("uri.String() = %q, want %q", uri.String(), tt.input)
				}
				return
			}
			if err == nil {
				t.Fatalf("NewBaseURI(%q) error = nil, want %q", tt.input, tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("NewBaseURI(%q) error = %q, want %q", tt.input, err.Error(), tt.wantErr)
			}
		})
	}
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
