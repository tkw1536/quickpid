package api_test

//spellchecker:words encoding json reflect strings testing github bicpid
import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/tkw1536/bicpid/api"
)

func TestSetNamespacePermissionRequest_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		body      string
		wantErr   bool
		wantErrIn []string
		want      api.SetNamespacePermissionRequest
	}{
		{
			name:    "ok_none",
			body:    `{"level":"none"}`,
			wantErr: false,
			want: api.SetNamespacePermissionRequest{
				Level: api.PermissionLevelNone,
			},
		},
		{
			name:    "ok_contributor",
			body:    `{"level":"contributor"}`,
			wantErr: false,
			want: api.SetNamespacePermissionRequest{
				Level: api.PermissionLevelContributor,
			},
		},
		{
			name:    "ok_editor",
			body:    `{"level":"editor"}`,
			wantErr: false,
			want: api.SetNamespacePermissionRequest{
				Level: api.PermissionLevelEditor,
			},
		},
		{
			name:    "ok_manager",
			body:    `{"level":"manager"}`,
			wantErr: false,
			want: api.SetNamespacePermissionRequest{
				Level: api.PermissionLevelManager,
			},
		},

		{
			name:      "fail_nullBody",
			body:      `null`,
			wantErr:   true,
			wantErrIn: []string{"expected JSON object"},
		},
		{
			name:      "fail_missingLevel",
			body:      `{}`,
			wantErr:   true,
			wantErrIn: []string{"missing required field", "level"},
		},
		{
			name:      "fail_unknownField",
			body:      `{"level":"editor","unknown":123}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields", "unknown field", "unknown"},
		},

		{
			name:      "fail_levelNull",
			body:      `{"level":null}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields"},
		},
		{
			name:      "fail_levelNumber",
			body:      `{"level":123}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields"},
		},
		{
			name:      "fail_levelBool",
			body:      `{"level":true}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields"},
		},
		{
			name:      "fail_levelObject",
			body:      `{"level":{}}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields"},
		},
		{
			name:      "fail_levelArray",
			body:      `{"level":[]}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields"},
		},

		{
			name:      "fail_levelEmptyString",
			body:      `{"level":""}`,
			wantErr:   true,
			wantErrIn: []string{"invalid permission level"},
		},
		{
			name:      "fail_invalidLevel",
			body:      `{"level":"admin"}`,
			wantErr:   true,
			wantErrIn: []string{"invalid permission level", "admin"},
		},
		{
			name:      "fail_invalidLevel_wrongCase",
			body:      `{"level":"Editor"}`,
			wantErr:   true,
			wantErrIn: []string{"invalid permission level", "Editor"},
		},
		{
			name:      "fail_invalidLevel_uppercase",
			body:      `{"level":"NONE"}`,
			wantErr:   true,
			wantErrIn: []string{"invalid permission level", "NONE"},
		},
		{
			name:      "fail_invalidLevel_whitespace",
			body:      `{"level":" contributor"}`,
			wantErr:   true,
			wantErrIn: []string{"invalid permission level", " contributor"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var req api.SetNamespacePermissionRequest
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
