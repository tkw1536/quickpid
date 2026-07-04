//spellchecker:words backend
package backend_test

//spellchecker:words testing github glebarez sqlite quickpid backend internal servertest gorm logger
import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/tkw1536/bicpid/backend"
	"github.com/tkw1536/bicpid/internal/servertest"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestGormBackend(t *testing.T) {
	servertest.TestBackend(t, func(t *testing.T) backend.Backend {
		t.Helper()
		db, err := gorm.Open(sqlite.Open(":memory:?_pragma=foreign_keys(1)"), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := backend.MigrateGorm(db); err != nil {
			t.Fatal(err)
		}
		return backend.NewGormBackend(db, 0)
	})
}
