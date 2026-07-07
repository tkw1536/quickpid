//spellchecker:words resolver
package resolver_test

//spellchecker:words testing github glebarez sqlite bicpid backend resolver internal servertest gorm logger
import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/tkw1536/bicpid/backend"
	"github.com/tkw1536/bicpid/backend/resolver"
	"github.com/tkw1536/bicpid/internal/servertest"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestGormBackend(t *testing.T) {
	servertest.TestBackend(t, func(t *testing.T) backend.ResolverBackend {
		t.Helper()
		db, err := gorm.Open(sqlite.Open(":memory:?_pragma=foreign_keys(1)"), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := resolver.MigrateGorm(db); err != nil {
			t.Fatal(err)
		}
		return resolver.NewGormBackend(db, 0)
	})
}
