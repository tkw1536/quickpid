//spellchecker:words memory
package memory_test

//spellchecker:words testing github quickpid backend memory storetest internal servertest
import (
	"testing"

	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/backend/memory"
	"github.com/tkw1536/quickpid/backend/storetest"
	"github.com/tkw1536/quickpid/internal/servertest"
)

func newStore() backend.Store {
	return memory.NewStore()
}

func TestStore_AuthUserCRUD(t *testing.T) {
	t.Parallel()

	storetest.RunAuthUserCRUD(t, func() backend.AuthenticationBackend { return newStore() })
}

func TestStore_AuthKeyLifecycle(t *testing.T) {
	t.Parallel()

	storetest.RunAuthKeyLifecycle(t, func() backend.AuthenticationBackend { return newStore() })
}

func TestStore_AuthNotFoundErrors(t *testing.T) {
	t.Parallel()

	storetest.RunAuthNotFoundErrors(t, func() backend.AuthenticationBackend { return newStore() })
}

func TestStore_AuthShutdown(t *testing.T) {
	t.Parallel()

	storetest.RunAuthShutdown(t, func() backend.AuthenticationBackend { return newStore() })
}

func TestStore_AuthListKeysSorted(t *testing.T) {
	t.Parallel()

	storetest.RunAuthListKeysSorted(t, func() backend.AuthenticationBackend { return newStore() })
}

func TestStore_AuthListUsers(t *testing.T) {
	t.Parallel()
	storetest.RunAuthListUsers(t, func() backend.AuthenticationBackend { return newStore() })
}

func TestStore_AuthAutocompleteUsers(t *testing.T) {
	t.Parallel()

	storetest.RunAuthAutocompleteUsers(t, func() backend.AuthenticationBackend { return newStore() })
}

func TestStore_AuthSuperuser(t *testing.T) {
	t.Parallel()

	storetest.RunAuthSuperuser(t, func() backend.AuthenticationBackend { return newStore() })
}

func TestStore_AuthorizationCRUD(t *testing.T) {
	t.Parallel()

	storetest.RunAuthorizationCRUD(t, newStore)
}

func TestStore_CreateNamespaceWithOwner(t *testing.T) {
	t.Parallel()

	storetest.RunCreateNamespaceWithOwner(t, newStore)
}

func TestStore_DeleteUserCascadesPermissions(t *testing.T) {
	t.Parallel()

	storetest.RunDeleteUserCascadesPermissions(t, newStore)
}

func TestStore_ResolverHTTP(t *testing.T) {
	t.Parallel()

	servertest.TestBackend(t, func(t *testing.T) backend.Store {
		t.Helper()
		return memory.NewStore()
	})
}
