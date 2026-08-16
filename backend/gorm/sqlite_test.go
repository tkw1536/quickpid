//spellchecker:words gorm
package gorm_test

//spellchecker:words slog testing github glebarez sqlite quickpid backend gorm gormbackend internal pidtest servertest logger Logger
import (
	"log/slog"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/tkw1536/quickpid/backend"
	gormbackend "github.com/tkw1536/quickpid/backend/gorm"
	servertest "github.com/tkw1536/quickpid/internal/pidtest"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

const sqliteDSN = ":memory:?_pragma=foreign_keys(1)"

func newSqliteBackend(t *testing.T, l *slog.Logger) backend.Backend {
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
	if err := gormbackend.Migrate(db); err != nil {
		t.Fatal(err)
	}
	return gormbackend.NewGormBackend(db, 0)
}

func TestGormBackend_sqlite(t *testing.T) {
	t.Parallel()

	servertest.RunBackendTests(t, newSqliteBackend, nil)
}

func TestGormBackend_Flows_sqlite(t *testing.T) {
	t.Parallel()

	servertest.RunFlowTests(t, newSqliteBackend, nil)
}
