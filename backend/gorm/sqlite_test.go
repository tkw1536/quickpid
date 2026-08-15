//spellchecker:words gorm
package gorm_test

//spellchecker:words testing github glebarez sqlite quickpid backend gorm gormstore internal pidtest servertest logger
import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/tkw1536/quickpid/backend"
	gormstore "github.com/tkw1536/quickpid/backend/gorm"
	servertest "github.com/tkw1536/quickpid/internal/pidtest"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newSqliteStore(t *testing.T) backend.Store {
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

func TestStore_sqlite(t *testing.T) {
	t.Parallel()

	servertest.RunStoreTests(t, newSqliteStore)
}

func TestStore_Flows_sqlite(t *testing.T) {
	t.Parallel()

	servertest.RunFlowTests(t, func(t *testing.T) backend.Store {
		t.Helper()
		return newSqliteStore(t)
	})
}
