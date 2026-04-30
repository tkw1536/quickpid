package spec_test

import (
	"encoding/json"
	"testing"

	"github.com/tkw1536/quickpid/spec"
)

func TestNamespaceCreateRequest_UnmarshalJSON_RequiresFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name:    "missingTag",
			body:    `{"pid_format":{"pattern":"***","characters":"full"}}`,
			wantErr: true,
		},
		{
			name:    "missingPIDFormat",
			body:    `{"tag":"ns"}`,
			wantErr: true,
		},
		{
			name:    "pidNotNullable",
			body:    `{"tag":null}`,
			wantErr: true,
		},
		{
			name:    "missingPIDFormatPattern",
			body:    `{"tag":"ns","pid_format":{"characters":"full"}}`,
			wantErr: true,
		},
		{
			name:    "missingPIDFormatCharacters",
			body:    `{"tag":"ns","pid_format":{"pattern":"***"}}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var req spec.NamespaceCreateRequest
			err := json.Unmarshal([]byte(tt.body), &req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error: got %v wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResourceCreateRequest_UnmarshalJSON_RequiresFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		body    string
		wantErr bool
		check   func(t *testing.T, got spec.ResourceCreateRequest)
	}{
		{
			name:    "missingURL",
			body:    `{"metadata":null,"tag":"t"}`,
			wantErr: true,
		},
		{
			name:    "missingTag",
			body:    `{"url":"https://example.com","metadata":null}`,
			wantErr: true,
		},
		{
			name:    "missingMetadata",
			body:    `{"url":"https://example.com","tag":"t"}`,
			wantErr: true,
		},
		{
			name:    "ok_metadataNull",
			body:    `{"url":"https://example.com","metadata":null,"tag":"t"}`,
			wantErr: false,
			check: func(t *testing.T, got spec.ResourceCreateRequest) {
				t.Helper()
				if got.Metadata != nil {
					t.Fatalf("metadata: got %v want nil", *got.Metadata)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var req spec.ResourceCreateRequest
			err := json.Unmarshal([]byte(tt.body), &req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error: got %v wantErr %v", err, tt.wantErr)
			}
			if err == nil && tt.check != nil {
				tt.check(t, req)
			}
		})
	}
}

func TestResourceUpdateRequest_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("url", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			body    string
			wantErr bool
			want    *string
		}{
			{name: "absent", body: `{}`, want: nil},
			{name: "string", body: `{"url":"https://example.com"}`, want: func() *string { s := "https://example.com"; return &s }()},
			{name: "emptyString", body: `{"url":""}`, want: func() *string { s := ""; return &s }()},
			{name: "null_isError", body: `{"url":null}`, wantErr: true},
			{name: "number_isError", body: `{"url":123}`, wantErr: true},
			{name: "bool_isError", body: `{"url":true}`, wantErr: true},
			{name: "object_isError", body: `{"url":{}}`, wantErr: true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				var req spec.ResourceUpdateRequest
				err := json.Unmarshal([]byte(tt.body), &req)
				if (err != nil) != tt.wantErr {
					t.Fatalf("error: got %v wantErr %v", err, tt.wantErr)
				}
				if err != nil {
					return
				}
				if (req.URL == nil) != (tt.want == nil) {
					t.Fatalf("url: got %v want %v", req.URL, tt.want)
				}
				if req.URL != nil && tt.want != nil && *req.URL != *tt.want {
					t.Fatalf("url: got %q want %q", *req.URL, *tt.want)
				}
			})
		}
	})

	t.Run("tag", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			body    string
			wantErr bool
			want    *string
		}{
			{name: "absent", body: `{}`, want: nil},
			{name: "string", body: `{"tag":"t"}`, want: func() *string { s := "t"; return &s }()},
			{name: "emptyString", body: `{"tag":""}`, want: func() *string { s := ""; return &s }()},
			{name: "null_isError", body: `{"tag":null}`, wantErr: true},
			{name: "number_isError", body: `{"tag":123}`, wantErr: true},
			{name: "bool_isError", body: `{"tag":true}`, wantErr: true},
			{name: "object_isError", body: `{"tag":{}}`, wantErr: true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				var req spec.ResourceUpdateRequest
				err := json.Unmarshal([]byte(tt.body), &req)
				if (err != nil) != tt.wantErr {
					t.Fatalf("error: got %v wantErr %v", err, tt.wantErr)
				}
				if err != nil {
					return
				}
				if (req.Tag == nil) != (tt.want == nil) {
					t.Fatalf("tag: got %v want %v", req.Tag, tt.want)
				}
				if req.Tag != nil && tt.want != nil && *req.Tag != *tt.want {
					t.Fatalf("tag: got %q want %q", *req.Tag, *tt.want)
				}
			})
		}
	})

	t.Run("metadata", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			body    string
			wantErr bool
			want    **string
		}{
			{name: "absent", body: `{}`, want: nil},
			{
				name: "null",
				body: `{"metadata":null}`,
				want: new((*string)(nil)),
			},
			{
				name: "string",
				body: `{"metadata":"m"}`,
				want: new(new("m")),
			},
			{name: "number_isError", body: `{"metadata":123}`, wantErr: true},
			{name: "bool_isError", body: `{"metadata":true}`, wantErr: true},
			{name: "object_isError", body: `{"metadata":{}}`, wantErr: true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				var req spec.ResourceUpdateRequest
				err := json.Unmarshal([]byte(tt.body), &req)
				if (err != nil) != tt.wantErr {
					t.Fatalf("error: got %v wantErr %v", err, tt.wantErr)
				}
				if err != nil {
					return
				}

				if (req.Metadata == nil) != (tt.want == nil) {
					t.Fatalf("metadata: got %v want %v", req.Metadata, tt.want)
				}
				if req.Metadata == nil || tt.want == nil {
					return
				}
				if (*req.Metadata == nil) != (*tt.want == nil) {
					t.Fatalf("metadata: got %v want %v", req.Metadata, tt.want)
				}
				if *req.Metadata != nil && *tt.want != nil && **req.Metadata != **tt.want {
					t.Fatalf("metadata: got %q want %q", **req.Metadata, **tt.want)
				}
			})
		}
	})

	t.Run("deleted", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			body    string
			wantErr bool
			want    *bool
		}{
			{name: "absent", body: `{}`, want: nil},
			{name: "true", body: `{"deleted":true}`, want: func() *bool { b := true; return &b }()},
			{name: "false", body: `{"deleted":false}`, want: func() *bool { b := false; return &b }()},
			{name: "null_isError", body: `{"deleted":null}`, wantErr: true},
			{name: "number_isError", body: `{"deleted":123}`, wantErr: true},
			{name: "string_isError", body: `{"deleted":"no"}`, wantErr: true},
			{name: "object_isError", body: `{"deleted":{}}`, wantErr: true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				var req spec.ResourceUpdateRequest
				err := json.Unmarshal([]byte(tt.body), &req)
				if (err != nil) != tt.wantErr {
					t.Fatalf("error: got %v wantErr %v", err, tt.wantErr)
				}
				if err != nil {
					return
				}
				if (req.Deleted == nil) != (tt.want == nil) {
					t.Fatalf("deleted: got %v want %v", req.Deleted, tt.want)
				}
				if req.Deleted != nil && tt.want != nil && *req.Deleted != *tt.want {
					t.Fatalf("deleted: got %v want %v", *req.Deleted, *tt.want)
				}
			})
		}
	})
}
