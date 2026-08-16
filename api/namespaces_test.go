package api_test

//spellchecker:words encoding json reflect strings testing github quickpid
import (
	"encoding/json/v2"
	"reflect"
	"strings"
	"testing"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/pid"
)

func TestNamespaceCreateRequest_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		body      string
		wantErr   bool
		wantErrIn []string
		want      api.NamespaceCreateRequest
	}{
		{
			name:    "ok",
			body:    `{"tags":["ns"],"pidFormat":{"pattern":"***","characters":"full"}}`,
			wantErr: false,
			want: api.NamespaceCreateRequest{
				Tags:      []string{"ns"},
				PIDFormat: pid.Format{Pattern: "***", Characters: pid.Full},
			},
		},
		{
			name:    "ok_emptyTags",
			body:    `{"tags":[],"pidFormat":{"pattern":"***","characters":"full"}}`,
			wantErr: false,
			want: api.NamespaceCreateRequest{
				Tags:      []string{},
				PIDFormat: pid.Format{Pattern: "***", Characters: pid.Full},
			},
		},
		{
			name:    "ok_multiTags",
			body:    `{"tags":["a","b"],"pidFormat":{"pattern":"***","characters":"full"}}`,
			wantErr: false,
			want: api.NamespaceCreateRequest{
				Tags:      []string{"a", "b"},
				PIDFormat: pid.Format{Pattern: "***", Characters: pid.Full},
			},
		},

		{
			name:      "fail_nullBody",
			body:      `null`,
			wantErr:   true,
			wantErrIn: []string{"json is null"},
		},

		{
			name:      "fail_missingTags",
			body:      `{"pidFormat":{"pattern":"***","characters":"full"}}`,
			wantErr:   true,
			wantErrIn: []string{"missing required field", "tags"},
		},
		{
			name:      "fail_unknownField",
			body:      `{"tags":["ns"],"pidFormat":{"pattern":"***","characters":"full"},"unknown":123}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields", "unknown object member name", "unknown"},
		},
		{
			name:      "fail_missingFormat",
			body:      `{"tags":["ns"]}`,
			wantErr:   true,
			wantErrIn: []string{"missing required field", "pidFormat"},
		},

		{
			name:      "fail_tagsNull",
			body:      `{"tags":null,"pidFormat":{"pattern":"***","characters":"full"}}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields"},
		},
		{
			name:      "fail_formatNull",
			body:      `{"tags":["ns"],"pidFormat":null}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields", "json is null"},
		},

		{
			name:      "fail_formatNull",
			body:      `{"tags":["ns"],"pidFormat":null}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields", "json is null"},
		},
		{
			name:      "fail_formatString",
			body:      `{"tags":["ns"],"pidFormat":"***"}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields", "failed to decode json"},
		},
		{
			name:      "fail_formatPattern",
			body:      `{"tags":["ns"],"pidFormat":{"characters":"full"}}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields", "missing required field", "pattern"},
		},
		{
			name:      "fail_formatCharactersMissing",
			body:      `{"tags":["ns"],"pidFormat":{"pattern":"***"}}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields", "missing required field", "characters"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var req api.NamespaceCreateRequest
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

func TestNamespaceUpdateRequest_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		body      string
		wantErr   bool
		wantErrIn []string
		want      api.NamespaceUpdateRequest
	}{
		{
			name:    "ok_empty",
			body:    `{}`,
			wantErr: false,
			want:    api.NamespaceUpdateRequest{},
		},
		{
			name:    "ok_tags",
			body:    `{"tags":["ns-updated"]}`,
			wantErr: false,
			want: api.NamespaceUpdateRequest{
				Tags: []string{"ns-updated"},
			},
		},
		{
			name:    "ok_tagsEmpty",
			body:    `{"tags":[]}`,
			wantErr: false,
			want: api.NamespaceUpdateRequest{
				Tags: []string{},
			},
		},
		{
			name:      "fail_nullBody",
			body:      `null`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal namespace update request", "json is null"},
		},
		{
			name:      "fail_unknownField",
			body:      `{"tags":["ns"],"unknown":123}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal namespace update request", "unknown object member name", "unknown"},
		},
		{
			name:      "fail_tagsNull",
			body:      `{"tags":null}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal namespace update request"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var req api.NamespaceUpdateRequest
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


func TestNewNamespaceID(t *testing.T) {
	t.Parallel()
	testIdentifierValidation(t, api.NewNamespaceID, "invalid namespace id")
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
