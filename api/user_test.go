package api_test

//spellchecker:words encoding json reflect strings testing github quickpid
import (
	"encoding/json/v2"
	"reflect"
	"strings"
	"testing"

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
			wantErrIn: []string{"json is null"},
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

func TestNewUsername(t *testing.T) {
	t.Parallel()
	testIdentifierValidation(t, api.NewUsername, "invalid username")
}

