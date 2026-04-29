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

func TestResourceUpdateRequest_UnmarshalJSON_RequiresFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		body    string
		wantErr bool
		check   func(t *testing.T, got spec.ResourceUpdateRequest)
	}{
		{
			name:    "missingDeleted",
			body:    `{"url":"https://example.com","metadata":null,"tag":"t"}`,
			wantErr: true,
		},
		{
			name:    "ok_deletedFalseExplicit",
			body:    `{"url":"https://example.com","metadata":null,"tag":"t","deleted":false}`,
			wantErr: false,
			check: func(t *testing.T, got spec.ResourceUpdateRequest) {
				t.Helper()
				if got.Deleted {
					t.Fatalf("deleted: got true want false")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var req spec.ResourceUpdateRequest
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
