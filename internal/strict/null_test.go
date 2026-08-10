//spellchecker:words strict
package strict_test

//spellchecker:words errors testing github quickpid internal strict
import (
	"errors"
	"testing"

	"github.com/tkw1536/quickpid/internal/strict"
)

func TestUnmarshalNonNull_errors(t *testing.T) {
	t.Parallel()

	type Payload struct {
		Name string `json:"name"`
	}

	tests := []struct {
		name    string
		body    string
		want    Payload
		wantErr error
	}{
		{
			name: "ok",
			body: `{"name":"alice"}`,
			want: Payload{Name: "alice"},
		},
		{
			name:    "fail_null",
			body:    `null`,
			wantErr: strict.ErrJsonNull,
		},
		{
			name:    "fail_array",
			body:    `[]`,
			wantErr: strict.ErrFailedToDecode,
		},
		{
			name:    "fail_string",
			body:    `"hello"`,
			wantErr: strict.ErrFailedToDecode,
		},
		{
			name:    "fail_unknownField",
			body:    `{"name":"alice","extra":1}`,
			wantErr: strict.ErrFailedToDecode,
		},
		{
			name:    "fail_trailingData",
			body:    `{"name":"alice"}trailing`,
			wantErr: strict.ErrTrailingData,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := strict.UnmarshalStrict[Payload]([]byte(tt.body))
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error: got %v, want errors.Is(..., %v)", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("error: got %v, want nil", err)
			}
			if got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestUnmarshalNonNull_success(t *testing.T) {
	t.Parallel()

	type Payload struct {
		Name string `json:"name"`
	}

	payload := Payload{Name: "alice"}
	got, err := strict.UnmarshalStrict[Payload]([]byte(`{"name":"alice"}`))
	if err != nil {
		t.Fatalf("error: got %v, want nil", err)
	}
	if got != payload {
		t.Fatalf("got %+v, want %+v", got, payload)
	}
}
