//spellchecker:words auth
package auth

//spellchecker:words crypto encoding base json errors http httptest testing golang bcrypt
import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// Known-good htpasswd hashes for password "secret".
const (
	//spellchecker:disable
	bcryptSecret = "$2y$05$y6RJtugGZOvBPuDg/f8.7O8GqTZAvUD8nNm1ipNrS8LiAm28fbdk2"
	apr1Secret   = "$apr1$abc123$i8aZw0pDl9Ea713mgwzXy1"
	//spellchecker:enable
)

func shaHash(password string) string {
	sum := sha1.Sum([]byte(password))
	return "{SHA}" + base64.StdEncoding.EncodeToString(sum[:])
}

func TestBasicAuthentication_Authenticate(t *testing.T) {
	t.Parallel()

	generatedBcrypt, err := bcrypt.GenerateFromPassword([]byte("bcryptpass"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt.GenerateFromPassword() error = %v", err)
	}

	defaultAuth := &BasicAuthentication{
		Users: []string{
			"alice:" + bcryptSecret,
			"bob:" + apr1Secret,
			"carol:" + shaHash("shapass"),
			"dave:" + string(generatedBcrypt),
		},
	}

	tests := []struct {
		name     string
		auth     *BasicAuthentication
		username string
		password string
		withAuth bool
		wantUser string
		wantErr  error
	}{
		{
			name:     "bcrypt",
			auth:     defaultAuth,
			username: "alice",
			password: "secret",
			withAuth: true,
			wantUser: "alice",
		},
		{
			name:     "apr1",
			auth:     defaultAuth,
			username: "bob",
			password: "secret",
			withAuth: true,
			wantUser: "bob",
		},
		{
			name:     "sha",
			auth:     defaultAuth,
			username: "carol",
			password: "shapass",
			withAuth: true,
			wantUser: "carol",
		},
		{
			name:     "generated bcrypt",
			auth:     defaultAuth,
			username: "dave",
			password: "bcryptpass",
			withAuth: true,
			wantUser: "dave",
		},
		{
			name:     "wrong password",
			auth:     defaultAuth,
			username: "alice",
			password: "wrong",
			withAuth: true,
			wantErr:  errUnauthorized,
		},
		{
			name:     "unknown user",
			auth:     defaultAuth,
			username: "eve",
			password: "secret",
			withAuth: true,
			wantErr:  errUnauthorized,
		},
		{
			name:    "missing authorization header",
			auth:    defaultAuth,
			wantErr: errUnauthorized,
		},
		{
			name:     "empty users",
			auth:     &BasicAuthentication{},
			username: "alice",
			password: "secret",
			withAuth: true,
			wantErr:  errUnauthorized,
		},
		{
			name: "unrecognized hash is ignored",
			auth: &BasicAuthentication{
				Users: []string{"alice:plaintext"},
			},
			username: "alice",
			password: "plaintext",
			withAuth: true,
			wantErr:  errUnauthorized,
		},
		{
			name: "malformed line does not block valid users",
			auth: &BasicAuthentication{
				Users: []string{
					"not-a-valid-line",
					"alice:" + bcryptSecret,
				},
			},
			username: "alice",
			password: "secret",
			withAuth: true,
			wantUser: "alice",
		},
		{
			name: "duplicate username uses last entry",
			auth: &BasicAuthentication{
				Users: []string{
					"alice:" + bcryptSecret,
					"alice:" + apr1Secret,
				},
			},
			username: "alice",
			password: "secret",
			withAuth: true,
			wantUser: "alice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.withAuth {
				req.SetBasicAuth(tt.username, tt.password)
			}

			gotUser, err := tt.auth.Authenticate(req)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if gotUser != tt.wantUser {
				t.Fatalf("username = %q, want %q", gotUser, tt.wantUser)
			}
		})
	}
}

func TestBasicAuthentication_MarshalJSON(t *testing.T) {
	t.Parallel()

	ba := &BasicAuthentication{
		Users: []string{
			"alice:" + bcryptSecret,
			"bob:" + apr1Secret,
		},
	}

	got, err := ba.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	want := `["alice:` + bcryptSecret + `","bob:` + apr1Secret + `"]`
	if string(got) != want {
		t.Fatalf("MarshalJSON() = %s, want %s", got, want)
	}
}

func TestBasicAuthentication_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	input := `["alice:` + bcryptSecret + `"]`
	var ba BasicAuthentication
	if err := ba.UnmarshalJSON([]byte(input)); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}

	if len(ba.Users) != 1 || ba.Users[0] != "alice:"+bcryptSecret {
		t.Fatalf("Users = %v, want %q", ba.Users, "alice:"+bcryptSecret)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("alice", "secret")
	gotUser, err := ba.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if gotUser != "alice" {
		t.Fatalf("username = %q, want %q", gotUser, "alice")
	}
}

func TestBasicAuthentication_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	original := &BasicAuthentication{
		Users: []string{"alice:" + bcryptSecret},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded BasicAuthentication
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(decoded.Users) != len(original.Users) || decoded.Users[0] != original.Users[0] {
		t.Fatalf("decoded Users = %v, want %v", decoded.Users, original.Users)
	}
}
