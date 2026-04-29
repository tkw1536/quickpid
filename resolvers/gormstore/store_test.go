package gormstore_test

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/internal/apitest"
	"github.com/tkw1536/quickpid/resolvers/gormstore"
	"github.com/tkw1536/quickpid/server"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestHTTP_ResolverFlow(t *testing.T) {
	apitest.RunResolverHTTPTests(t, func(t *testing.T) backend.ResolverBackend {
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
		return gormstore.NewResolver(db, server.DefaultPIDMaxAttempts)
	})
}
