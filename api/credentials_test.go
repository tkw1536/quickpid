package api_test

//spellchecker:words encoding json reflect strings testing time github quickpid
import (
	"encoding/json/v2"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/tkw1536/quickpid/api"
)

func TestSetPasswordRequest_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		body      string
		wantErr   bool
		wantErrIn []string
		wantNil   bool
		wantValue string
	}{
		{name: "string", body: `{"password":"secret"}`, wantValue: "secret"},
		{name: "null", body: `{"password":null}`, wantNil: true},
		{name: "missing", body: `{}`, wantErr: true, wantErrIn: []string{"missing required field", "password"}},
		{name: "unknown", body: `{"password":"secret","unknown":1}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields", "unknown object member name", "unknown"}},
		{name: "number", body: `{"password":123}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var req api.SetPasswordRequest
			err := json.Unmarshal([]byte(tt.body), &req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error: got %v wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				es := err.Error()
				for _, wantIn := range tt.wantErrIn {
					if !strings.Contains(es, wantIn) {
						t.Fatalf("error: got %q want substring %q", es, wantIn)
					}
				}
				return
			}
			if tt.wantNil {
				if req.Password != nil {
					t.Fatalf("req.Password = %v, want nil payload", *req.Password)
				}
				return
			}
			if req.Password == nil || *req.Password != tt.wantValue {
				t.Fatalf("req.Password = %v, want %q", req.Password, tt.wantValue)
			}
		})
	}
}

func TestSetPasswordRequest_Validate(t *testing.T) {
	t.Parallel()

	t.Run("string", func(t *testing.T) {
		t.Parallel()

		req := api.SetPasswordRequest{Password: new("secret")}
		valid, err := req.Validate()
		if err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		if valid.Password == nil || valid.Password.String() != "secret" {
			t.Fatalf("Validate() = %+v, want password secret", valid)
		}
	})

	t.Run("null", func(t *testing.T) {
		t.Parallel()

		req := api.SetPasswordRequest{Password: (*string)(nil)}
		valid, err := req.Validate()
		if err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		if valid.Password != nil {
			t.Fatalf("Validate() = %+v, want nil password", valid)
		}
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		req := api.SetPasswordRequest{Password: new("")}
		_, err := req.Validate()
		if err == nil {
			t.Fatal("Validate() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "invalid password") {
			t.Fatalf("Validate() error = %q, want invalid password", err.Error())
		}
	})
}

func TestKeyIssueRequest_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		body      string
		wantErr   bool
		wantErrIn []string
		want      api.KeyIssueRequest
	}{
		{
			name:      "fail_missingExpiresAt",
			body:      `{"comment":"dev key"}`,
			wantErr:   true,
			wantErrIn: []string{"missing required field", "expiresAt"},
		},
		{
			name:    "ok_expiresAtSet",
			body:    `{"comment":"dev key","expiresAt":"2026-12-31T00:00:00Z"}`,
			wantErr: false,
			want: api.KeyIssueRequest{
				Comment:   "dev key",
				ExpiresAt: new(time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)),
			},
		},
		{
			name:      "fail_usernameInBody",
			body:      `{"comment":"dev key","expiresAt":"2026-12-31T00:00:00Z","username":"alice"}`,
			wantErr:   true,
			wantErrIn: []string{"unknown object member name", "username"},
		},
		{
			name:      "fail_nullBody",
			body:      `null`,
			wantErr:   true,
			wantErrIn: []string{"json is null"},
		},
		{
			name:      "fail_missingComment",
			body:      `{}`,
			wantErr:   true,
			wantErrIn: []string{"missing required field", "comment"},
		},
		{
			name:      "fail_unknownField",
			body:      `{"comment":"dev key","expiresAt":null,"unknown":123}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields", "unknown object member name", "unknown"},
		},
		{
			name:      "fail_commentNull",
			body:      `{"comment":null,"expiresAt":null}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields"},
		},
		{
			name:    "ok_expiresAtNull",
			body:    `{"comment":"dev key","expiresAt":null}`,
			wantErr: false,
			want: api.KeyIssueRequest{
				Comment:   "dev key",
				ExpiresAt: nil,
			},
		},
		{
			name:      "fail_expiresAtMalformed",
			body:      `{"comment":"dev key","expiresAt":"not-a-time"}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields"},
		},
		{
			name:      "fail_usernameNull",
			body:      `{"comment":"dev key","expiresAt":null,"username":null}`,
			wantErr:   true,
			wantErrIn: []string{"unknown object member name", "username"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var req api.KeyIssueRequest
			err := json.Unmarshal([]byte(tt.body), &req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error: got %v wantErr %v", err, tt.wantErr)
			}
			if err != nil && len(tt.wantErrIn) > 0 {
				es := err.Error()
				for _, wantIn := range tt.wantErrIn {
					if !strings.Contains(es, wantIn) {
						t.Fatalf("error: got %q want substring %q", es, wantIn)
					}
				}
			}
			if err == nil && !tt.wantErr && !reflect.DeepEqual(req, tt.want) {
				t.Fatalf("req: got %+v want %+v", req, tt.want)
			}
		})
	}
}

func TestKeyRevokeRequest_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		body      string
		wantErr   bool
		wantErrIn []string
		want      api.KeyRevokeRequest
	}{
		{
			name:    "ok",
			body:    `{"id":"key-123"}`,
			wantErr: false,
			want: api.KeyRevokeRequest{
				ID: "key-123",
			},
		},
		{
			name:      "fail_nullBody",
			body:      `null`,
			wantErr:   true,
			wantErrIn: []string{"json is null"},
		},
		{
			name:      "fail_missingID",
			body:      `{}`,
			wantErr:   true,
			wantErrIn: []string{"missing required field", "id"},
		},
		{
			name:      "fail_unknownField",
			body:      `{"id":"key-123","unknown":123}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields", "unknown object member name", "unknown"},
		},
		{
			name:      "fail_idNull",
			body:      `{"id":null}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var req api.KeyRevokeRequest
			err := json.Unmarshal([]byte(tt.body), &req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error: got %v wantErr %v", err, tt.wantErr)
			}
			if err != nil && len(tt.wantErrIn) > 0 {
				es := err.Error()
				for _, wantIn := range tt.wantErrIn {
					if !strings.Contains(es, wantIn) {
						t.Fatalf("error: got %q want substring %q", es, wantIn)
					}
				}
			}
			if err == nil && !tt.wantErr && !reflect.DeepEqual(req, tt.want) {
				t.Fatalf("req: got %+v want %+v", req, tt.want)
			}
		})
	}
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

