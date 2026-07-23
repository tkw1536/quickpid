package api_test

//spellchecker:words encoding json reflect strings testing github quickpid
import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/tkw1536/quickpid/api"
)

func TestSetNamespaceRoleRequest_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		body      string
		wantErr   bool
		wantErrIn []string
		want      api.SetNamespaceRoleRequest
	}{
		{
			name:    "ok_none",
			body:    `{"role":"none"}`,
			wantErr: false,
			want: api.SetNamespaceRoleRequest{
				Role: api.RoleNone,
			},
		},
		{
			name:    "ok_contributor",
			body:    `{"role":"contributor"}`,
			wantErr: false,
			want: api.SetNamespaceRoleRequest{
				Role: api.RoleContributor,
			},
		},
		{
			name:    "ok_editor",
			body:    `{"role":"editor"}`,
			wantErr: false,
			want: api.SetNamespaceRoleRequest{
				Role: api.RoleEditor,
			},
		},
		{
			name:    "ok_manager",
			body:    `{"role":"manager"}`,
			wantErr: false,
			want: api.SetNamespaceRoleRequest{
				Role: api.RoleManager,
			},
		},

		{
			name:      "fail_nullBody",
			body:      `null`,
			wantErr:   true,
			wantErrIn: []string{"expected JSON object"},
		},
		{
			name:      "fail_missingRole",
			body:      `{}`,
			wantErr:   true,
			wantErrIn: []string{"missing required field", "role"},
		},
		{
			name:      "fail_unknownField",
			body:      `{"role":"editor","unknown":123}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields", "unknown field", "unknown"},
		},

		{
			name:      "fail_roleNull",
			body:      `{"role":null}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields"},
		},
		{
			name:      "fail_roleNumber",
			body:      `{"role":123}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields"},
		},
		{
			name:      "fail_roleBool",
			body:      `{"role":true}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields"},
		},
		{
			name:      "fail_roleObject",
			body:      `{"role":{}}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields"},
		},
		{
			name:      "fail_roleArray",
			body:      `{"role":[]}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields"},
		},

		{
			name:      "fail_roleEmptyString",
			body:      `{"role":""}`,
			wantErr:   true,
			wantErrIn: []string{"invalid role"},
		},
		{
			name:      "fail_invalidRole",
			body:      `{"role":"admin"}`,
			wantErr:   true,
			wantErrIn: []string{"invalid role", "admin"},
		},
		{
			name:      "fail_invalidRole_wrongCase",
			body:      `{"role":"Editor"}`,
			wantErr:   true,
			wantErrIn: []string{"invalid role", "Editor"},
		},
		{
			name:      "fail_invalidRole_uppercase",
			body:      `{"role":"NONE"}`,
			wantErr:   true,
			wantErrIn: []string{"invalid role", "NONE"},
		},
		{
			name:      "fail_invalidRole_whitespace",
			body:      `{"role":" contributor"}`,
			wantErr:   true,
			wantErrIn: []string{"invalid role", " contributor"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var req api.SetNamespaceRoleRequest
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
