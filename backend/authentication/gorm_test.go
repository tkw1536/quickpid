//spellchecker:words authentication
package authentication_test

//spellchecker:words testing github glebarez sqlite bicpid backend authentication gorm logger
import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/tkw1536/bicpid/backend"
	"github.com/tkw1536/bicpid/backend/authentication"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newGormBackend(t *testing.T) backend.AuthBackend {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:?_pragma=foreign_keys(1)"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := authentication.MigrateGorm(db); err != nil {
		t.Fatal(err)
	}
	return authentication.NewGormBackend(db)
}

func TestGormBackend_UserCRUD(t *testing.T) {
	testBackendUserCRUD(t, func() backend.AuthBackend { return newGormBackend(t) })
}

func TestGormBackend_KeyLifecycle(t *testing.T) {
	testBackendKeyLifecycle(t, func() backend.AuthBackend { return newGormBackend(t) })
}

func TestGormBackend_NotFoundErrors(t *testing.T) {
	testBackendNotFoundErrors(t, func() backend.AuthBackend { return newGormBackend(t) })
}

func TestGormBackend_KeyNotStoredInPlaintext(t *testing.T) {
	testBackendKeyNotStoredInPlaintext(t, func() backend.AuthBackend { return newGormBackend(t) })
}

func TestGormBackend_Shutdown(t *testing.T) {
	testBackendShutdown(t, func() backend.AuthBackend { return newGormBackend(t) })
}

func TestGormBackend_ListKeysSorted(t *testing.T) {
	testBackendListKeysSorted(t, func() backend.AuthBackend { return newGormBackend(t) })
}

func TestGormBackend_ListUsers(t *testing.T) {
	testBackendListUsers(t, func() backend.AuthBackend { return newGormBackend(t) })
}

func TestGormBackend_UpdateKeyExpiresAt(t *testing.T) {
	testBackendUpdateKeyExpiresAt(t, func() backend.AuthBackend { return newGormBackend(t) })
}

func TestGormBackend_Superuser(t *testing.T) {
	testBackendSuperuser(t, func() backend.AuthBackend { return newGormBackend(t) })
}
