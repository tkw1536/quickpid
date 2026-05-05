package main

import (
	"cmp"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/glebarez/sqlite"
	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/cmd"
	"gorm.io/gorm"
)

func main() {
	cmd.Main("quickpid-sqlite", func(logger *slog.Logger) (backend.Backend, error) {
		logger.Info("opening database", "dsn", sqliteDSN)
		db, err := gorm.Open(sqlite.Open(sqliteDSN), &gorm.Config{})
		if err != nil {
			return nil, fmt.Errorf("failed to open database: %w", err)
		}
		if !disableAutoMigrate {
			if err := backend.MigrateGorm(db); err != nil {
				return nil, fmt.Errorf("failed to migrate database: %w", err)
			}
		}
		return backend.NewGormBackend(db, 0), nil
	})
}

var (
	sqliteDSN          string = cmp.Or(os.Getenv("DSN"), "quickpid.db?_pragma=foreign_keys(1)")
	disableAutoMigrate bool   = false
)

func init() {
	flag.StringVar(&sqliteDSN, "dsn", sqliteDSN, "SQLite database connection string (can also be set via DSN environment variable)")
	flag.BoolVar(&disableAutoMigrate, "disable-auto-migrate", disableAutoMigrate, "disable automatic database migration")
}
