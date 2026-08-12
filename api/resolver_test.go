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
			wantErrIn: []string{"json is null"},
		},

		// url
		{
			name:      "fail_missingURL",
			body:      `{"metadata":"m","tags":["t"]}`,
			wantErr:   true,
			wantErrIn: []string{"missing required field", "url"},
		},
		{
			name:      "fail_unknownField",
			body:      `{"url":"https://example.com","metadata":null,"tags":["t"],"unknown":123}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields", "unknown object member name", "unknown"},
		},
		{
			name:    "ok_urlNull",
			body:    `{"url":null,"metadata":"m","tags":["t"]}`,
			wantErr: false,
			want: api.ResourceCreateRequest{
				URL:      nil,
				Metadata: new("m"),
				Tags:     []string{"t"},
			},
		},

		// tags
		{
			name:      "fail_missingTags",
			body:      `{"url":"https://example.com","metadata":"m"}`,
			wantErr:   true,
			wantErrIn: []string{"missing required field", "tags"},
		},
		{
			name:      "fail_singularTag",
			body:      `{"url":"https://example.com","metadata":null,"tag":"t"}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields", "unknown object member name", "tag"},
		},
		{
			name:      "fail_singularTagWithTags",
			body:      `{"url":"https://example.com","metadata":null,"tag":"a","tags":["b","c"]}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields", "unknown object member name", "tag"},
		},
		{
			name:      "fail_tagsNull",
			body:      `{"url":"https://example.com","metadata":null,"tags":null}`,
			wantErr:   true,
			wantErrIn: []string{"failed to unmarshal fields"},
		},

		// metadata
		{
			name:      "fail_missingMetadata",
			body:      `{"url":"https://example.com","tags":["t"]}`,
			wantErr:   true,
			wantErrIn: []string{"missing required field", "metadata"},
		},
		{
			name:    "ok_tagsEmpty",
			body:    `{"url":"https://example.com","metadata":null,"tags":[]}`,
			wantErr: false,
			want: api.ResourceCreateRequest{
				URL:      new("https://example.com"),
				Metadata: nil,
				Tags:     []string{},
			},
		},
		{
			name:    "ok_tagsOneItem",
			body:    `{"url":"https://example.com","metadata":null,"tags":["t"]}`,
			wantErr: false,
			want: api.ResourceCreateRequest{
				URL:      new("https://example.com"),
				Metadata: nil,
				Tags:     []string{"t"},
			},
		},
		{
			name:    "ok_metadataString",
			body:    `{"url":"https://example.com","metadata":"m","tags":["t"]}`,
			wantErr: false,
			want: api.ResourceCreateRequest{
				URL:      new("https://example.com"),
				Metadata: new("m"),
				Tags:     []string{"t"},
			},
		},
		{
			name:    "ok_multiTags",
			body:    `{"url":"https://example.com","metadata":null,"tags":["a","b"]}`,
			wantErr: false,
			want: api.ResourceCreateRequest{
				URL:      new("https://example.com"),
				Metadata: nil,
				Tags:     []string{"a", "b"},
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
			{name: "absent", body: `{}`, want: api.ResourceUpdateRequest{URL: nil, Tags: nil, Deleted: nil, Metadata: nil}},
			{name: "string", body: `{"url":"https://example.com"}`, want: api.ResourceUpdateRequest{URL: new(new("https://example.com")), Tags: nil, Deleted: nil, Metadata: nil}},
			{name: "emptyString", body: `{"url":""}`, want: api.ResourceUpdateRequest{URL: new(new("")), Tags: nil, Deleted: nil, Metadata: nil}},
			{name: "null", body: `{"url":null}`, want: api.ResourceUpdateRequest{URL: new(*string), Tags: nil, Deleted: nil, Metadata: nil}},
			{name: "number_isError", body: `{"url":123}`, wantErr: true, wantErrIn: []string{"failed to decode json"}},
			{name: "bool_isError", body: `{"url":true}`, wantErr: true, wantErrIn: []string{"failed to decode json"}},
			{name: "object_isError", body: `{"url":{}}`, wantErr: true, wantErrIn: []string{"failed to decode json"}},
			{name: "unknownField_isError", body: `{"url":"https://example.com","unknown":123}`, wantErr: true, wantErrIn: []string{"failed to decode json", "unknown object member name", "unknown"}},
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

	t.Run("tags", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name      string
			body      string
			wantErr   bool
			wantErrIn []string
			want      api.ResourceUpdateRequest
		}{
			{name: "absent", body: `{}`, want: api.ResourceUpdateRequest{URL: nil, Tags: nil, Deleted: nil, Metadata: nil}},
			{name: "empty", body: `{"tags":[]}`, want: api.ResourceUpdateRequest{URL: nil, Tags: []string{}, Deleted: nil, Metadata: nil}},
			{name: "oneItem", body: `{"tags":["t"]}`, want: api.ResourceUpdateRequest{URL: nil, Tags: []string{"t"}, Deleted: nil, Metadata: nil}},
			{name: "emptyStringItem", body: `{"tags":[""]}`, want: api.ResourceUpdateRequest{URL: nil, Tags: []string{""}, Deleted: nil, Metadata: nil}},
			{name: "multiTags", body: `{"tags":["a","b"]}`, want: api.ResourceUpdateRequest{URL: nil, Tags: []string{"a", "b"}, Deleted: nil, Metadata: nil}},
			{name: "singularTag_isError", body: `{"tag":"t"}`, wantErr: true, wantErrIn: []string{"failed to decode json", "unknown object member name", "tag"}},
			{name: "singularTagWithTags_isError", body: `{"tag":"a","tags":["b","c"]}`, wantErr: true, wantErrIn: []string{"failed to decode json", "unknown object member name", "tag"}},
			{name: "null_isError", body: `{"tags":null}`, wantErr: true, wantErrIn: []string{"failed to decode json"}},
			{name: "number_isError", body: `{"tags":123}`, wantErr: true, wantErrIn: []string{"failed to decode json"}},
			{name: "bool_isError", body: `{"tags":true}`, wantErr: true, wantErrIn: []string{"failed to decode json"}},
			{name: "object_isError", body: `{"tags":{}}`, wantErr: true, wantErrIn: []string{"failed to decode json"}},
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
			{name: "absent", body: `{}`, want: api.ResourceUpdateRequest{URL: nil, Tags: nil, Deleted: nil, Metadata: nil}},
			{
				name: "null",
				body: `{"metadata":null}`,
				want: api.ResourceUpdateRequest{URL: nil, Tags: nil, Deleted: nil, Metadata: new(*string)},
			},
			{
				name: "string",
				body: `{"metadata":"m"}`,
				want: api.ResourceUpdateRequest{URL: nil, Tags: nil, Deleted: nil, Metadata: new(new("m"))},
			},
			{name: "number_isError", body: `{"metadata":123}`, wantErr: true, wantErrIn: []string{"failed to decode json"}},
			{name: "bool_isError", body: `{"metadata":true}`, wantErr: true, wantErrIn: []string{"failed to decode json"}},
			{name: "object_isError", body: `{"metadata":{}}`, wantErr: true, wantErrIn: []string{"failed to decode json"}},
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
			{name: "absent", body: `{}`, want: api.ResourceUpdateRequest{URL: nil, Tags: nil, Deleted: nil, Metadata: nil}},
			{name: "true", body: `{"deleted":true}`, want: api.ResourceUpdateRequest{URL: nil, Tags: nil, Deleted: new(true), Metadata: nil}},
			{name: "false", body: `{"deleted":false}`, want: api.ResourceUpdateRequest{URL: nil, Tags: nil, Deleted: new(false), Metadata: nil}},
			{name: "null_isError", body: `{"deleted":null}`, wantErr: true, wantErrIn: []string{"failed to decode json"}},
			{name: "number_isError", body: `{"deleted":123}`, wantErr: true, wantErrIn: []string{"failed to decode json"}},
			{name: "string_isError", body: `{"deleted":"no"}`, wantErr: true, wantErrIn: []string{"failed to decode json"}},
			{name: "object_isError", body: `{"deleted":{}}`, wantErr: true, wantErrIn: []string{"failed to decode json"}},
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

func TestResourceCreateRequest_Validate(t *testing.T) {
	t.Parallel()

	t.Run("validURL", func(t *testing.T) {
		t.Parallel()
		req := api.ResourceCreateRequest{
			URL:      new("https://example.com/path?q=1#frag"),
			Metadata: nil,
			Tags:     []string{"t"},
		}
		got, err := req.Validate()
		if err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		if got.URL == nil || got.URL.String() != "https://example.com/path?q=1#frag" {
			t.Fatalf("URL = %v, want validated https URL", got.URL)
		}
	})

	t.Run("nullURL", func(t *testing.T) {
		t.Parallel()
		req := api.ResourceCreateRequest{URL: nil, Metadata: new("m"), Tags: []string{}}
		got, err := req.Validate()
		if err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		if got.URL != nil {
			t.Fatalf("URL = %v, want nil", got.URL)
		}
	})

	t.Run("invalidURL", func(t *testing.T) {
		t.Parallel()
		req := api.ResourceCreateRequest{URL: new("not-a-uri"), Metadata: nil, Tags: []string{}}
		_, err := req.Validate()
		if err == nil {
			t.Fatal("Validate() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "not an absolute URI") {
			t.Fatalf("Validate() error = %q, want absolute URI error", err.Error())
		}
	})
}

func TestResourceUpdateRequest_Validate(t *testing.T) {
	t.Parallel()

	t.Run("absentURL", func(t *testing.T) {
		t.Parallel()
		req := api.ResourceUpdateRequest{}
		got, err := req.Validate()
		if err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		if got.URL != nil {
			t.Fatalf("URL = %v, want nil", got.URL)
		}
	})

	t.Run("nullURL", func(t *testing.T) {
		t.Parallel()
		req := api.ResourceUpdateRequest{URL: new(*string)}
		got, err := req.Validate()
		if err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		if got.URL == nil || *got.URL != nil {
			t.Fatalf("URL = %v, want &nil", got.URL)
		}
	})

	t.Run("validURL", func(t *testing.T) {
		t.Parallel()
		req := api.ResourceUpdateRequest{URL: new(new("https://example.com"))}
		got, err := req.Validate()
		if err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		if got.URL == nil || *got.URL == nil || (*got.URL).String() != "https://example.com" {
			t.Fatalf("URL = %v, want validated https URL", got.URL)
		}
	})

	t.Run("invalidURL", func(t *testing.T) {
		t.Parallel()
		req := api.ResourceUpdateRequest{URL: new(new("/relative"))}
		_, err := req.Validate()
		if err == nil {
			t.Fatal("Validate() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "not an absolute URI") {
			t.Fatalf("Validate() error = %q, want absolute URI error", err.Error())
		}
	})
}
