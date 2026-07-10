//spellchecker:words memory
package memory_test

//spellchecker:words testing github bicpid backend memory storetest internal servertest
import (
	"testing"

	"github.com/tkw1536/bicpid/backend"
	"github.com/tkw1536/bicpid/backend/memory"
	"github.com/tkw1536/bicpid/backend/storetest"
	"github.com/tkw1536/bicpid/internal/servertest"
)

func newStore() backend.Store {
	return memory.NewStore()
}

func TestStore_AuthUserCRUD(t *testing.T) {
	storetest.RunAuthUserCRUD(t, func() backend.AuthenticationBackend { return newStore() })
}

func TestStore_AuthKeyLifecycle(t *testing.T) {
	storetest.RunAuthKeyLifecycle(t, func() backend.AuthenticationBackend { return newStore() })
}

func TestStore_AuthNotFoundErrors(t *testing.T) {
	storetest.RunAuthNotFoundErrors(t, func() backend.AuthenticationBackend { return newStore() })
}

func TestStore_AuthShutdown(t *testing.T) {
	storetest.RunAuthShutdown(t, func() backend.AuthenticationBackend { return newStore() })
}

func TestStore_AuthListKeysSorted(t *testing.T) {
	storetest.RunAuthListKeysSorted(t, func() backend.AuthenticationBackend { return newStore() })
}

func TestStore_AuthListUsers(t *testing.T) {
	storetest.RunAuthListUsers(t, func() backend.AuthenticationBackend { return newStore() })
}

func TestStore_AuthSuperuser(t *testing.T) {
	storetest.RunAuthSuperuser(t, func() backend.AuthenticationBackend { return newStore() })
}

func TestStore_AuthorizationCRUD(t *testing.T) {
	storetest.RunAuthorizationCRUD(t, newStore)
}

func TestStore_CreateNamespaceWithOwner(t *testing.T) {
	storetest.RunCreateNamespaceWithOwner(t, newStore)
}

func TestStore_DeleteUserCascadesPermissions(t *testing.T) {
	storetest.RunDeleteUserCascadesPermissions(t, newStore)
}

func TestStore_ResolverHTTP(t *testing.T) {
	servertest.TestBackend(t, func(t *testing.T) backend.Store {
		t.Helper()
		return memory.NewStore()
	})
}
