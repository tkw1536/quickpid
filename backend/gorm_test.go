package backend_test

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/internal/apitest"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestGormBackend(t *testing.T) {
	apitest.RunResolverHTTPTests(t, func(t *testing.T) backend.Backend {
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
		return backend.NewGormBackend(db)
	})
}
