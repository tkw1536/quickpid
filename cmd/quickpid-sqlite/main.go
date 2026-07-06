//spellchecker:words main
package main

//spellchecker:words flag slog github glebarez sqlite quickpid backend authentication gorm
import (
	"cmp"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/glebarez/sqlite"
	"github.com/tkw1536/bicpid/backend"
	"github.com/tkw1536/bicpid/backend/authentication"
	"github.com/tkw1536/bicpid/backend/resolver"
	"github.com/tkw1536/bicpid/cmd"
	"gorm.io/gorm"
)

func main() {
	var db *gorm.DB
	cmd.Main("quickpid-sqlite",
		func(logger *slog.Logger) error {
			logger.Info("opening database", "dsn", sqliteDSN)
			var err error
			db, err = gorm.Open(sqlite.Open(sqliteDSN), &gorm.Config{})
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			return nil
		},
		func(logger *slog.Logger) (backend.ResolverBackend, error) {
			if !disableAutoMigrate {
				if err := resolver.MigrateGorm(db); err != nil {
					return nil, fmt.Errorf("failed to migrate database: %w", err)
				}
			}
			return resolver.NewGormBackend(db, 0), nil
		},
		func(logger *slog.Logger) (backend.AuthBackend, error) {
			if !disableAutoMigrate {
				if err := authentication.MigrateGorm(db); err != nil {
					return nil, fmt.Errorf("failed to migrate auth database: %w", err)
				}
			}
			return authentication.NewGormBackend(db), nil
		},
	)
}

var (
	sqliteDSN          string = cmp.Or(os.Getenv("DSN"), "quickpid.db?_pragma=foreign_keys(1)")
	disableAutoMigrate bool   = false
)

func init() {
	flag.StringVar(&sqliteDSN, "dsn", sqliteDSN, "SQLite database connection string (can also be set via DSN environment variable)")
	flag.BoolVar(&disableAutoMigrate, "disable-auto-migrate", disableAutoMigrate, "disable automatic database migration")
}
