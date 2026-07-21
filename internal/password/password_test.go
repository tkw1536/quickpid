//spellchecker:words password
package password_test

//spellchecker:words bytes testing github quickpid internal
import (
	"bytes"
	"testing"

	"github.com/tkw1536/quickpid/internal/password"
)

func TestHashVerify(t *testing.T) {
	t.Parallel()

	const secret = "correct horse battery staple"

	hash, err := password.Hash(secret)
	if err != nil {
		t.Fatalf("password.Hash() error = %v", err)
	}
	if len(hash) == 0 {
		t.Fatal("password.Hash() returned empty hash")
	}
	if bytes.Equal(hash, []byte(secret)) {
		t.Fatal("password.Hash() returned plaintext bytes")
	}
	if !password.Verify(secret, hash) {
		t.Fatal("password.Verify() = false, want true")
	}
	if password.Verify("wrong password", hash) {
		t.Fatal("password.Verify() = true for wrong password, want false")
	}
}
