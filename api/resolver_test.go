package api_test

//spellchecker:words encoding json reflect strings testing github quickpid
import (
	"encoding/json"
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
			body:    `{"tag":"ns","pid_format":{"pattern":"***","characters":"full"}}`,
			wantErr: false,
			want: api.NamespaceCreateRequest{
				Tag:       "ns",
				PIDFormat: pid.Format{Pattern: "***", Characters: pid.Full},
			},
		},

		{
			name:      "fail_nullBody",
			body:      `null`,
			wantErr:   true,
			wantErrIn: []string{"expected JSON object"},
		},

		{
			name:      "fail_missingTag",
			body:      `{"pid_format":{"pattern":"***","characters":"full"}}`,
			wantErr:   true,
			wantErrIn: []string{"missing required field", "tag"},
		},
		{
			name:      "fail_unknownField",
			body:      `{"tag":"ns","pid_format":{"pattern":"***","characters":"full"},"unknown":123}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields", "unknown field", "unknown"},
		},
		{
			name:      "fail_missingFormat",
			body:      `{"tag":"ns"}`,
			wantErr:   true,
			wantErrIn: []string{"missing required field", "pid_format"},
		},

		{
			name:      "fail_tagNull",
			body:      `{"tag":null,"pid_format":{"pattern":"***","characters":"full"}}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields"},
		},
		{
			name:      "fail_formatNull",
			body:      `{"tag":"ns","pid_format":null}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields", "expected JSON object"},
		},

		{
			name:      "fail_formatNull",
			body:      `{"tag":"ns","pid_format":null}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields", "expected JSON object"},
		},
		{
			name:      "fail_formatString",
			body:      `{"tag":"ns","pid_format":"***"}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields", "expected JSON object"},
		},
		{
			name:      "fail_formatPattern",
			body:      `{"tag":"ns","pid_format":{"characters":"full"}}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields", "missing required field", "pattern"},
		},
		{
			name:      "fail_formatCharactersMissing",
			body:      `{"tag":"ns","pid_format":{"pattern":"***"}}`,
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

func TestResourceCreateRequest_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		body      string
		wantErr   bool
		wantErrIn []string
		want      api.ResourceCreateRequest
	}{
		{
			name:      "fail_null",
			body:      `null`,
			wantErr:   true,
			wantErrIn: []string{"expected JSON object"},
		},

		// url
		{
			name:      "fail_missingURL",
			body:      `{"metadata":"m","tag":"t"}`,
			wantErr:   true,
			wantErrIn: []string{"missing required field", "url"},
		},
		{
			name:      "fail_unknownField",
			body:      `{"url":"https://example.com","metadata":null,"tag":"t","unknown":123}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields", "unknown field", "unknown"},
		},
		{
			name:      "fail_urlNull",
			body:      `{"url":null,"metadata":"m","tag":"t"}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields"},
		},

		// tag
		{
			name:      "fail_missingTag",
			body:      `{"url":"https://example.com","metadata":"m"}`,
			wantErr:   true,
			wantErrIn: []string{"missing required field", "tag"},
		},
		{
			name:      "fail_tagNull",
			body:      `{"url":"https://example.com","metadata":null,"tag":null}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields"},
		},

		// metadata
		{
			name:      "fail_missingMetadata",
			body:      `{"url":"https://example.com","tag":"t"}`,
			wantErr:   true,
			wantErrIn: []string{"missing required field", "metadata"},
		},
		{
			name:    "ok_metadataNull",
			body:    `{"url":"https://example.com","metadata":null,"tag":"t"}`,
			wantErr: false,
			want: api.ResourceCreateRequest{
				URL:      "https://example.com",
				Metadata: nil,
				Tag:      "t",
			},
		},
		{
			name:    "ok_metadataString",
			body:    `{"url":"https://example.com","metadata":"m","tag":"t"}`,
			wantErr: false,
			want: api.ResourceCreateRequest{
				URL:      "https://example.com",
				Metadata: new("m"),
				Tag:      "t",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var req api.ResourceCreateRequest
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

func TestResourceUpdateRequest_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("url", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name      string
			body      string
			wantErr   bool
			wantErrIn []string
			want      api.ResourceUpdateRequest
		}{
			{name: "absent", body: `{}`, want: api.ResourceUpdateRequest{URL: nil, Tag: nil, Deleted: nil, Metadata: nil}},
			{name: "string", body: `{"url":"https://example.com"}`, want: api.ResourceUpdateRequest{URL: new("https://example.com"), Tag: nil, Deleted: nil, Metadata: nil}},
			{name: "emptyString", body: `{"url":""}`, want: api.ResourceUpdateRequest{URL: new(""), Tag: nil, Deleted: nil, Metadata: nil}},
			{name: "null_isError", body: `{"url":null}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
			{name: "number_isError", body: `{"url":123}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
			{name: "bool_isError", body: `{"url":true}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
			{name: "object_isError", body: `{"url":{}}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
			{name: "unknownField_isError", body: `{"url":"https://example.com","unknown":123}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields", "unknown field", "unknown"}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				var req api.ResourceUpdateRequest
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

	t.Run("tag", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name      string
			body      string
			wantErr   bool
			wantErrIn []string
			want      api.ResourceUpdateRequest
		}{
			{name: "absent", body: `{}`, want: api.ResourceUpdateRequest{URL: nil, Tag: nil, Deleted: nil, Metadata: nil}},
			{name: "string", body: `{"tag":"t"}`, want: api.ResourceUpdateRequest{URL: nil, Tag: new("t"), Deleted: nil, Metadata: nil}},
			{name: "emptyString", body: `{"tag":""}`, want: api.ResourceUpdateRequest{URL: nil, Tag: new(""), Deleted: nil, Metadata: nil}},
			{name: "null_isError", body: `{"tag":null}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
			{name: "number_isError", body: `{"tag":123}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
			{name: "bool_isError", body: `{"tag":true}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
			{name: "object_isError", body: `{"tag":{}}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				var req api.ResourceUpdateRequest
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

	t.Run("metadata", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name      string
			body      string
			wantErr   bool
			wantErrIn []string
			want      api.ResourceUpdateRequest
		}{
			{name: "absent", body: `{}`, want: api.ResourceUpdateRequest{URL: nil, Tag: nil, Deleted: nil, Metadata: nil}},
			{
				name: "null",
				body: `{"metadata":null}`,
				want: api.ResourceUpdateRequest{URL: nil, Tag: nil, Deleted: nil, Metadata: new(*string)},
			},
			{
				name: "string",
				body: `{"metadata":"m"}`,
				want: api.ResourceUpdateRequest{URL: nil, Tag: nil, Deleted: nil, Metadata: new(new("m"))},
			},
			{name: "number_isError", body: `{"metadata":123}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
			{name: "bool_isError", body: `{"metadata":true}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
			{name: "object_isError", body: `{"metadata":{}}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				var req api.ResourceUpdateRequest
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

	t.Run("deleted", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name      string
			body      string
			wantErr   bool
			wantErrIn []string
			want      api.ResourceUpdateRequest
		}{
			{name: "absent", body: `{}`, want: api.ResourceUpdateRequest{URL: nil, Tag: nil, Deleted: nil, Metadata: nil}},
			{name: "true", body: `{"deleted":true}`, want: api.ResourceUpdateRequest{URL: nil, Tag: nil, Deleted: new(true), Metadata: nil}},
			{name: "false", body: `{"deleted":false}`, want: api.ResourceUpdateRequest{URL: nil, Tag: nil, Deleted: new(false), Metadata: nil}},
			{name: "null_isError", body: `{"deleted":null}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
			{name: "number_isError", body: `{"deleted":123}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
			{name: "string_isError", body: `{"deleted":"no"}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
			{name: "object_isError", body: `{"deleted":{}}`, wantErr: true, wantErrIn: []string{"failed to unmarshal fields"}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				var req api.ResourceUpdateRequest
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
