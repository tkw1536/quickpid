package api_test

//spellchecker:words testing github quickpid
import (
	"testing"

	"github.com/tkw1536/quickpid/api"
)

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
			wantErr: "not an absolute URI",
		},
		{
			name:    "invalid_relative",
			input:   "/relative/path",
			wantErr: "not an absolute URI",
		},
		{
			name:    "invalid_no_scheme",
			input:   "example.com/foo",
			wantErr: "not an absolute URI",
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

