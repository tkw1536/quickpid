//spellchecker:words gorm
package gorm_test

//spellchecker:words testing github glebarez sqlite quickpid backend gorm gormstore storetest internal servertest logger
import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/tkw1536/quickpid/backend"
	gormstore "github.com/tkw1536/quickpid/backend/gorm"
	"github.com/tkw1536/quickpid/backend/storetest"
	"github.com/tkw1536/quickpid/internal/servertest"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newStore(t *testing.T) backend.Store {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:?_pragma=foreign_keys(1)"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := gormstore.Migrate(db); err != nil {
		t.Fatal(err)
	}
	return gormstore.NewStore(db, 0)
}

func TestStore_AuthUserCRUD(t *testing.T) {
	t.Parallel()

	storetest.RunAuthUserCRUD(t, func() backend.AuthenticationBackend { return newStore(t) })
}

func TestStore_AuthKeyLifecycle(t *testing.T) {
	t.Parallel()

	storetest.RunAuthKeyLifecycle(t, func() backend.AuthenticationBackend { return newStore(t) })
}

func TestStore_AuthNotFoundErrors(t *testing.T) {
	t.Parallel()

	storetest.RunAuthNotFoundErrors(t, func() backend.AuthenticationBackend { return newStore(t) })
}

func TestStore_AuthShutdown(t *testing.T) {
	t.Parallel()

	storetest.RunAuthShutdown(t, func() backend.AuthenticationBackend { return newStore(t) })
}

func TestStore_AuthListKeysSorted(t *testing.T) {
	t.Parallel()

	storetest.RunAuthListKeysSorted(t, func() backend.AuthenticationBackend { return newStore(t) })
}

func TestStore_AuthListUsers(t *testing.T) {
	t.Parallel()

	storetest.RunAuthListUsers(t, func() backend.AuthenticationBackend { return newStore(t) })
}

func TestStore_AuthAutocompleteUsers(t *testing.T) {
	t.Parallel()

	storetest.RunAuthAutocompleteUsers(t, func() backend.AuthenticationBackend { return newStore(t) })
}

func TestStore_AuthSuperuser(t *testing.T) {
	t.Parallel()

	storetest.RunAuthSuperuser(t, func() backend.AuthenticationBackend { return newStore(t) })
}

func TestStore_AuthorizationCRUD(t *testing.T) {
	t.Parallel()

	storetest.RunAuthorizationCRUD(t, func() backend.Store { return newStore(t) })
}

func TestStore_CreateNamespaceWithOwner(t *testing.T) {
	t.Parallel()

	storetest.RunCreateNamespaceWithOwner(t, func() backend.Store { return newStore(t) })
}

func TestStore_DeleteUserCascadesPermissions(t *testing.T) {
	t.Parallel()

	storetest.RunDeleteUserCascadesPermissions(t, func() backend.Store { return newStore(t) })
}

func TestStore_ResolverHTTP(t *testing.T) {
	t.Parallel()

	servertest.TestBackend(t, func(t *testing.T) backend.Store {
		t.Helper()
		return newStore(t)
	})
}
