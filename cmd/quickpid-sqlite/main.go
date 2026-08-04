//spellchecker:words main
package main

//spellchecker:words flag slog github glebarez sqlite quickpid backend gorm gormstore
import (
	"cmp"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/glebarez/sqlite"
	"github.com/tkw1536/quickpid/backend"
	gormstore "github.com/tkw1536/quickpid/backend/gorm"
	"github.com/tkw1536/quickpid/cmd"
	"gorm.io/gorm"
)

var errForeignKeysNotEnabled = errors.New("SQlite foreign keys are not enabled. You have to enable them via '?_pragma=foreign_keys(1)' in the connection DSN")

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
			enabled, err := gormstore.SQLiteForeignKeysEnabled(db)
			if err != nil {
				return fmt.Errorf("failed to check sqlite foreign keys: %w", err)
			}
			if !enabled {
				return errForeignKeysNotEnabled
			}
			return nil
		},
		func(logger *slog.Logger) (backend.Store, error) {
			if !disableAutoMigrate {
				if err := gormstore.Migrate(db); err != nil {
					return nil, fmt.Errorf("failed to migrate database: %w", err)
				}
			}
			return gormstore.NewStore(db, 0), nil
		},
	)
}

var (
	sqliteDSN          string = cmp.Or(os.Getenv("DSN"), "quickpid.db?_pragma=foreign_keys(1)")
	disableAutoMigrate bool   = false
)

func init() {
	flag.StringVar(&sqliteDSN, "dsn", sqliteDSN, "SQLite database connection string (can also be set via DSN environment variable). You have to enable foreign keys via ?_pragma=foreign_keys(1) in the DSN for this to work properly.")
	flag.BoolVar(&disableAutoMigrate, "disable-auto-migrate", disableAutoMigrate, "disable automatic database migration")
}
