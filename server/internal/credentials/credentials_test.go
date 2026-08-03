//spellchecker:words credentials
package credentials_test

//spellchecker:words encoding base http testing github quickpid server internal credentials
import (
	"encoding/base64"
	"fmt"
	"net/http"
	"testing"

	"github.com/tkw1536/quickpid/server/internal/credentials"
)

func ExampleParse() {
	h := make(http.Header)
	h.Set("Authorization", "Bearer secret-token")

	creds := credentials.Parse(h)
	fmt.Println(creds.Kind() == credentials.KindBearer)
	fmt.Println(creds.BearerToken())
	// Output: true
	// secret-token
}

func TestParse(t *testing.T) {
	t.Parallel()

	const (
		token     = "secret-token"
		username  = "alice"
		password  = "s3cr3t"
		basicUser = "alice:s3cr3t"
	)
	basicValue := "Basic " + base64.StdEncoding.EncodeToString([]byte(basicUser))
	basicNoColon := "Basic " + base64.StdEncoding.EncodeToString([]byte("alice"))
	basicBadB64 := "Basic $$$"

	tests := []struct {
		name string
		auth []string

		wantKind credentials.Kind

		wantBearerToken string

		wantBasicUsername string
		wantBasicPassword string
	}{
		{
			name:     "no Authorization header",
			auth:     nil,
			wantKind: credentials.KindEmpty,
		},
		{
			name:            "single Bearer",
			auth:            []string{"Bearer " + token},
			wantKind:        credentials.KindBearer,
			wantBearerToken: token,
		},
		{
			name:            "single bearer lowercase scheme",
			auth:            []string{"bearer " + token},
			wantKind:        credentials.KindBearer,
			wantBearerToken: token,
		},
		{
			name:            "single BEARER uppercase scheme",
			auth:            []string{"BEARER " + token},
			wantKind:        credentials.KindBearer,
			wantBearerToken: token,
		},
		{
			name:            "single Bearer trims token whitespace",
			auth:            []string{"Bearer  " + token + "  "},
			wantKind:        credentials.KindBearer,
			wantBearerToken: token,
		},
		{
			name:            "single Bearer with empty token",
			auth:            []string{"Bearer "},
			wantKind:        credentials.KindBearer,
			wantBearerToken: "",
		},
		{
			name:              "single Basic",
			auth:              []string{basicValue},
			wantKind:          credentials.KindBasic,
			wantBasicUsername: username,
			wantBasicPassword: password,
		},
		{
			name:              "single basic lowercase scheme",
			auth:              []string{"basic " + basicValue[len("Basic "):]},
			wantKind:          credentials.KindBasic,
			wantBasicUsername: username,
			wantBasicPassword: password,
		},
		{
			name:              "single BASIC uppercase scheme",
			auth:              []string{"BASIC " + basicValue[len("Basic "):]},
			wantKind:          credentials.KindBasic,
			wantBasicUsername: username,
			wantBasicPassword: password,
		},
		{
			name:     "Bearer without space after scheme",
			auth:     []string{"Bearer" + token},
			wantKind: credentials.KindInvalid,
		},
		{
			name:     "Basic without space after scheme",
			auth:     []string{"Basic" + basicValue[len("Basic "):]},
			wantKind: credentials.KindInvalid,
		},
		{
			name:     "empty Authorization value",
			auth:     []string{""},
			wantKind: credentials.KindInvalid,
		},
		{
			name:     "unsupported Digest scheme",
			auth:     []string{`Digest username="alice"`},
			wantKind: credentials.KindInvalid,
		},
		{
			name:     "unsupported Token scheme",
			auth:     []string{"Token " + token},
			wantKind: credentials.KindInvalid,
		},
		{
			name:     "Basic with invalid base64",
			auth:     []string{basicBadB64},
			wantKind: credentials.KindInvalid,
		},
		{
			name:     "Basic with decoded value missing colon",
			auth:     []string{basicNoColon},
			wantKind: credentials.KindInvalid,
		},
		{
			name:     "two Bearer headers",
			auth:     []string{"Bearer " + token, "Bearer other-token"},
			wantKind: credentials.KindInvalid,
		},
		{
			name:     "two identical Bearer headers",
			auth:     []string{"Bearer " + token, "Bearer " + token},
			wantKind: credentials.KindInvalid,
		},
		{
			name:     "two Basic headers",
			auth:     []string{basicValue, basicValue},
			wantKind: credentials.KindInvalid,
		},
		{
			name:     "Bearer then Basic",
			auth:     []string{"Bearer " + token, basicValue},
			wantKind: credentials.KindInvalid,
		},
		{
			name:     "Basic then Bearer",
			auth:     []string{basicValue, "Bearer " + token},
			wantKind: credentials.KindInvalid,
		},
		{
			name:     "Bearer then unsupported scheme",
			auth:     []string{"Bearer " + token, `Digest username="alice"`},
			wantKind: credentials.KindInvalid,
		},
		{
			name:     "unsupported scheme then Bearer",
			auth:     []string{`Digest username="alice"`, "Bearer " + token},
			wantKind: credentials.KindInvalid,
		},
		{
			name:     "Basic then unsupported scheme",
			auth:     []string{basicValue, `Digest username="alice"`},
			wantKind: credentials.KindInvalid,
		},
		{
			name:     "three Bearer headers",
			auth:     []string{"Bearer one", "Bearer two", "Bearer three"},
			wantKind: credentials.KindInvalid,
		},
		{
			name:     "Bearer, Basic, and Bearer",
			auth:     []string{"Bearer " + token, basicValue, "Bearer other"},
			wantKind: credentials.KindInvalid,
		},
		{
			name:     "empty value among otherwise valid Bearer",
			auth:     []string{"Bearer " + token, ""},
			wantKind: credentials.KindInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := credentials.Parse(authHeader(tt.auth...))

			if kind := got.Kind(); kind != tt.wantKind {
				t.Errorf("Kind() = %v, want %v", kind, tt.wantKind)
			}

			switch tt.wantKind {
			case credentials.KindBearer:
				if token := got.BearerToken(); token != tt.wantBearerToken {
					t.Errorf("BearerToken() = %q, want %q", token, tt.wantBearerToken)
				}
				assertPanics(t, "BasicAuth", func() { got.BasicAuth() })
			case credentials.KindBasic:
				username, password := got.BasicAuth()
				if username != tt.wantBasicUsername || password != tt.wantBasicPassword {
					t.Errorf(
						"BasicAuth() = (%q, %q), want (%q, %q)",
						username, password,
						tt.wantBasicUsername, tt.wantBasicPassword,
					)
				}
				assertPanics(t, "BearerToken", func() { got.BearerToken() })
			case credentials.KindInvalid, credentials.KindEmpty:
				assertPanics(t, "BearerToken", func() { got.BearerToken() })
				assertPanics(t, "BasicAuth", func() { got.BasicAuth() })
			}
		})
	}
}

func assertPanics(t *testing.T, name string, fn func()) {
	t.Helper()

	defer func() {
		if recover() == nil {
			t.Errorf("%s() did not panic", name)
		}
	}()
	fn()
}

func authHeader(values ...string) http.Header {
	h := make(http.Header)
	for _, value := range values {
		h.Add("Authorization", value)
	}
	return h
}
