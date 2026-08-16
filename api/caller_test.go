package api_test

//spellchecker:words encoding json testing time github quickpid
import (
	"encoding/json/v2"
	"testing"
	"time"

	"github.com/tkw1536/quickpid/api"
)

func TestAuthenticationMethod_MarshalJSON(t *testing.T) {
	t.Parallel()

	alice, err := api.NewUsername("alice")
	if err != nil {
		t.Fatalf("NewUsername() error = %v", err)
	}

	expiresAt := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		method api.AuthenticationMethod
		want   string
	}{
		{
			name:   "none",
			method: api.NoAuthentication{},
			want:   `{"type":"none"}`,
		},
		{
			name:   "basic",
			method: api.BasicAuthentication{},
			want:   `{"type":"basic"}`,
		},
		{
			name: "token",
			method: api.TokenAuthentication{
				Token: api.APIKeyInfo{
					ID:        "key-123",
					Comment:   "test key",
					CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
					ExpiresAt: &expiresAt,
				},
			},
			want: `{"type":"token"}`,
		},
		{
			name: "impersonate",
			method: api.Impersonation{
				User: alice,
				Reason: api.TokenAuthentication{
					Token: api.APIKeyInfo{ID: "key-123"},
				},
			},
			want: `{"type":"impersonate","user":"alice","reason":{"type":"token"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := json.Marshal(tt.method)
			if err != nil {
				t.Fatalf("MarshalJSON() error = %v", err)
			}
			if string(got) != tt.want {
				t.Fatalf("MarshalJSON() = %s, want %s", got, tt.want)
			}
		})
	}
}
