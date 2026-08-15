//spellchecker:words gorm
package gorm_test

//spellchecker:words slog testing github glebarez sqlite quickpid backend gorm gormstore internal pidtest servertest logger Logger
import (
	"log/slog"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/tkw1536/quickpid/backend"
	gormstore "github.com/tkw1536/quickpid/backend/gorm"
	servertest "github.com/tkw1536/quickpid/internal/pidtest"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

const sqliteDSN = ":memory:?_pragma=foreign_keys(1)"

func newSqliteStore(t *testing.T, l *slog.Logger) backend.Store {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(sqliteDSN), &gorm.Config{
		Logger: gormLogger.NewSlogLogger(l, gormLogger.Config{
			LogLevel:                  gormLogger.Info,
			IgnoreRecordNotFoundError: true,
		}),
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

	servertest.RunStoreTests(t, newSqliteStore, nil)
}

func TestStore_Flows_sqlite(t *testing.T) {
	t.Parallel()

	servertest.RunFlowTests(t, newSqliteStore, nil)
}
