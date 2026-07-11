package api_test

//spellchecker:words encoding json reflect strings testing github bicpid
import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/tkw1536/bicpid/api"
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
			wantErrIn: []string{"failed to unmarshal fields", "unknown field", "unknown"},
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

	t.Run("username", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name      string
			body      string
			wantErr   bool
			wantErrIn []string
			want      api.UserUpdateRequest
		}{
			{name: "absent", body: `{}`, want: api.UserUpdateRequest{Username: nil, Superuser: nil}},
			{name: "string", body: `{"username":"alice"}`, want: api.UserUpdateRequest{Username: new("alice"), Superuser: nil}},
			{name: "emptyString", body: `{"username":""}`, want: api.UserUpdateRequest{Username: new(""), Superuser: nil}},
			{name: "null_isError", body: `{"username":null}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
			{name: "number_isError", body: `{"username":123}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
			{name: "bool_isError", body: `{"username":true}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
			{name: "object_isError", body: `{"username":{}}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
			{name: "unknownField_isError", body: `{"username":"alice","unknown":123}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields", "unknown field", "unknown"}},
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

	t.Run("superuser", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name      string
			body      string
			wantErr   bool
			wantErrIn []string
			want      api.UserUpdateRequest
		}{
			{name: "absent", body: `{}`, want: api.UserUpdateRequest{Username: nil, Superuser: nil}},
			{name: "true", body: `{"superuser":true}`, want: api.UserUpdateRequest{Username: nil, Superuser: new(true)}},
			{name: "false", body: `{"superuser":false}`, want: api.UserUpdateRequest{Username: nil, Superuser: new(false)}},
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
			name:    "ok_commentOnly",
			body:    `{"comment":"dev key"}`,
			wantErr: false,
			want: api.KeyIssueRequest{
				Comment:   "dev key",
				ExpiresAt: nil,
				Username:  nil,
			},
		},
		{
			name:    "ok_allFields",
			body:    `{"comment":"dev key","expires_at":"2026-12-31T00:00:00Z","username":"alice"}`,
			wantErr: false,
			want: api.KeyIssueRequest{
				Comment:   "dev key",
				ExpiresAt: new("2026-12-31T00:00:00Z"),
				Username:  new("alice"),
			},
		},
		{
			name:      "fail_nullBody",
			body:      `null`,
			wantErr:   true,
			wantErrIn: []string{"expected JSON object"},
		},
		{
			name:      "fail_missingComment",
			body:      `{"username":"alice"}`,
			wantErr:   true,
			wantErrIn: []string{"missing required field", "comment"},
		},
		{
			name:      "fail_unknownField",
			body:      `{"comment":"dev key","unknown":123}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields", "unknown field", "unknown"},
		},
		{
			name:      "fail_commentNull",
			body:      `{"comment":null}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields"},
		},
		{
			name:    "ok_expiresAtNull",
			body:    `{"comment":"dev key","expires_at":null}`,
			wantErr: false,
			want: api.KeyIssueRequest{
				Comment:   "dev key",
				ExpiresAt: nil,
				Username:  nil,
			},
		},
		{
			name:      "fail_usernameNull",
			body:      `{"comment":"dev key","username":null}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields"},
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
			wantErrIn: []string{"failed to unmarshal fields", "unknown field", "unknown"},
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

func TestUpdateKeyRequest_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("comment", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name      string
			body      string
			wantErr   bool
			wantErrIn []string
			want      api.KeyUpdateRequest
		}{
			{name: "absent", body: `{}`, want: api.KeyUpdateRequest{Comment: nil, ExpiresAt: nil}},
			{name: "string", body: `{"comment":"updated"}`, want: api.KeyUpdateRequest{Comment: new("updated"), ExpiresAt: nil}},
			{name: "emptyString", body: `{"comment":""}`, want: api.KeyUpdateRequest{Comment: new(""), ExpiresAt: nil}},
			{name: "null_isError", body: `{"comment":null}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
			{name: "number_isError", body: `{"comment":123}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
			{name: "bool_isError", body: `{"comment":true}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
			{name: "object_isError", body: `{"comment":{}}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
			{name: "unknownField_isError", body: `{"comment":"updated","unknown":123}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields", "unknown field", "unknown"}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				var req api.KeyUpdateRequest
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

	t.Run("expires_at", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name      string
			body      string
			wantErr   bool
			wantErrIn []string
			want      api.KeyUpdateRequest
		}{
			{name: "absent", body: `{}`, want: api.KeyUpdateRequest{Comment: nil, ExpiresAt: nil}},
			{
				name: "null",
				body: `{"expires_at":null}`,
				want: api.KeyUpdateRequest{Comment: nil, ExpiresAt: new(*string)},
			},
			{
				name: "string",
				body: `{"expires_at":"2026-12-31T00:00:00Z"}`,
				want: api.KeyUpdateRequest{Comment: nil, ExpiresAt: new(new("2026-12-31T00:00:00Z"))},
			},
			{name: "number_isError", body: `{"expires_at":123}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
			{name: "bool_isError", body: `{"expires_at":true}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
			{name: "object_isError", body: `{"expires_at":{}}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				var req api.KeyUpdateRequest
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
