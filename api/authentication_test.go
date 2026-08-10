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

func TestUserCreateRequest_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		body      string
		wantErr   bool
		wantErrIn []string
		want      api.UserCreateRequest
	}{
		{
			name:    "ok",
			body:    `{"username":"alice","superuser":true}`,
			wantErr: false,
			want: api.UserCreateRequest{
				Username:  "alice",
				Superuser: true,
			},
		},
		{
			name:    "ok_superuserAbsent",
			body:    `{"username":"alice"}`,
			wantErr: false,
			want: api.UserCreateRequest{
				Username:  "alice",
				Superuser: false,
			},
		},
		{
			name:      "fail_nullBody",
			body:      `null`,
			wantErr:   true,
			wantErrIn: []string{"expected JSON object"},
		},
		{
			name:      "fail_missingUsername",
			body:      `{"superuser":true}`,
			wantErr:   true,
			wantErrIn: []string{"missing required field", "username"},
		},
		{
			name:      "fail_unknownField",
			body:      `{"username":"alice","superuser":true,"unknown":123}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields", "unknown object member name", "unknown"},
		},
		{
			name:      "fail_usernameNull",
			body:      `{"username":null,"superuser":true}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields"},
		},
		{
			name:      "fail_superuserNull",
			body:      `{"username":"alice","superuser":null}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields"},
		},
		{
			name:      "fail_superuserString",
			body:      `{"username":"alice","superuser":"yes"}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var req api.UserCreateRequest
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

func TestUserUpdateRequest_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("username_in_body_is_rejected", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name      string
			body      string
			wantErrIn []string
		}{
			{name: "string", body: `{"username":"alice"}`, wantErrIn: []string{"unknown object member name", "username"}},
			{name: "emptyString", body: `{"username":""}`, wantErrIn: []string{"unknown object member name", "username"}},
			{name: "null", body: `{"username":null}`, wantErrIn: []string{"unknown object member name", "username"}},
			{name: "number", body: `{"username":123}`, wantErrIn: []string{"unknown object member name", "username"}},
			{name: "bool", body: `{"username":true}`, wantErrIn: []string{"unknown object member name", "username"}},
			{name: "object", body: `{"username":{}}`, wantErrIn: []string{"unknown object member name", "username"}},
			{name: "unknownField", body: `{"username":"alice","unknown":123}`, wantErrIn: []string{"unknown object member name"}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				var req api.UserUpdateRequest
				err := json.Unmarshal([]byte(tt.body), &req)
				if err == nil {
					t.Fatalf("error: got nil want error")
				}
				es := err.Error()
				for _, wantIn := range tt.wantErrIn {
					if !strings.Contains(es, wantIn) {
						t.Fatalf("error: got %q want substring %q", es, wantIn)
					}
				}
			})
		}
	})

	t.Run("superuser", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name      string
			body      string
			wantErr   bool
			wantErrIn []string
			want      api.UserUpdateRequest
		}{
			{name: "absent", body: `{}`, want: api.UserUpdateRequest{Superuser: nil}},
			{name: "true", body: `{"superuser":true}`, want: api.UserUpdateRequest{Superuser: new(true)}},
			{name: "false", body: `{"superuser":false}`, want: api.UserUpdateRequest{Superuser: new(false)}},
			{name: "null_isError", body: `{"superuser":null}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
			{name: "number_isError", body: `{"superuser":123}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
			{name: "string_isError", body: `{"superuser":"yes"}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
			{name: "object_isError", body: `{"superuser":{}}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				var req api.UserUpdateRequest
				err := json.Unmarshal([]byte(tt.body), &req)
				if (err != nil) != tt.wantErr {
					t.Fatalf("error: got %v wantErr %v", err, tt.wantErr)
				}
				if err != nil {
					if len(tt.wantErrIn) > 0 {
						es := err.Error()
						for _, wantIn := range tt.wantErrIn {
							if !strings.Contains(es, wantIn) {
								t.Fatalf("error: got %q want substring %q", es, wantIn)
							}
						}
					}
					return
				}
				if !reflect.DeepEqual(req, tt.want) {
					t.Fatalf("req: got %+v want %+v", req, tt.want)
				}
			})
		}
	})
}

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
			if req.Password == nil {
				t.Fatal("req.Password = nil, want non-nil field marker")
			}
			if tt.wantNil {
				if *req.Password != nil {
					t.Fatalf("req.Password = %v, want nil payload", **req.Password)
				}
				return
			}
			if *req.Password == nil || **req.Password != tt.wantValue {
				t.Fatalf("req.Password = %v, want %q", req.Password, tt.wantValue)
			}
		})
	}
}

func TestSetPasswordRequest_Validate(t *testing.T) {
	t.Parallel()

	t.Run("string", func(t *testing.T) {
		t.Parallel()

		value := "secret"
		ptr := &value
		req := api.SetPasswordRequest{Password: &ptr}
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

		var ptr *string
		req := api.SetPasswordRequest{Password: &ptr}
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

		value := ""
		ptr := &value
		req := api.SetPasswordRequest{Password: &ptr}
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
			wantErrIn: []string{"expected JSON object"},
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
			wantErrIn: []string{"expected JSON object"},
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
