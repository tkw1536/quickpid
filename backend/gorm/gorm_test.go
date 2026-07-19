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

func TestStore(t *testing.T) {
	t.Parallel()

	servertest.RunStoreTests(t, newStore)
}

func TestStore_ResolverHTTP(t *testing.T) {
	t.Parallel()

	servertest.RunServerTests(t, func(t *testing.T) backend.Store {
		t.Helper()
		return newStore(t)
	})
}
