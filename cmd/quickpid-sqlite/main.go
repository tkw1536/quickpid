package main

import (
	"flag"
	"fmt"

	"github.com/glebarez/sqlite"
	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/cmd"
	"gorm.io/gorm"
)

func main() {
	cmd.Main(func() (backend.Backend, error) {
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
	sqliteDSN          string = "quickpid.db?_pragma=foreign_keys(1)"
	disableAutoMigrate bool   = false
)

func init() {
	flag.StringVar(&sqliteDSN, "sqlite-dsn", sqliteDSN, "SQLite database connection string")
	flag.BoolVar(&disableAutoMigrate, "disable-auto-migrate", disableAutoMigrate, "disable automatic database migration")
}
