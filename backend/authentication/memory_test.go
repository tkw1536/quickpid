//spellchecker:words backend authentication
package authentication_test

//spellchecker:words strings testing time github quickpid backend authentication
import (
	"strings"
	"testing"
	"time"

	"github.com/tkw1536/bicpid/backend"
	"github.com/tkw1536/bicpid/backend/authentication"
)

// testAPIKey returns a 32-character lowercase alphanumeric key for use in tests.
func testAPIKey(suffix string) string {
	if len(suffix) >= 32 {
		return suffix[:32]
	}
	return strings.Repeat("a", 32-len(suffix)) + suffix
}

func fixedNow() func() time.Time {
	t := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	return func() time.Time { return t }
}

func newInMemoryBackend() backend.AuthBackend {
	return authentication.NewInMemoryBackend()
}

func TestInMemoryBackend_UserCRUD(t *testing.T) {
	testBackendUserCRUD(t, newInMemoryBackend)
}

func TestInMemoryBackend_KeyLifecycle(t *testing.T) {
	testBackendKeyLifecycle(t, newInMemoryBackend)
}

func TestInMemoryBackend_NotFoundErrors(t *testing.T) {
	testBackendNotFoundErrors(t, newInMemoryBackend)
}

func TestInMemoryBackend_KeyNotStoredInPlaintext(t *testing.T) {
	testBackendKeyNotStoredInPlaintext(t, newInMemoryBackend)
}

func TestInMemoryBackend_Shutdown(t *testing.T) {
	testBackendShutdown(t, newInMemoryBackend)
}

func TestInMemoryBackend_ListKeysSorted(t *testing.T) {
	testBackendListKeysSorted(t, newInMemoryBackend)
}

func TestInMemoryBackend_ListUsers(t *testing.T) {
	testBackendListUsers(t, newInMemoryBackend)
}

func TestInMemoryBackend_UpdateKeyExpiresAt(t *testing.T) {
	testBackendUpdateKeyExpiresAt(t, newInMemoryBackend)
}

func TestInMemoryBackend_Superuser(t *testing.T) {
	testBackendSuperuser(t, newInMemoryBackend)
}

func ptrStringPtr(v *string) **string {
	return &v
}
