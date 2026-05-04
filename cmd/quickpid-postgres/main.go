package main

import (
	"flag"
	"fmt"

	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/cmd"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cmd.Main("quickpid-postgres", func() (backend.Backend, error) {
		db, err := gorm.Open(postgres.Open(postgresDSN), &gorm.Config{})
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
	// spin up a temporary local postgres with something like:
	// docker run --rm -e POSTGRES_PASSWORD=quickpid -e POSTGRES_DB=quickpid -p 5432:5432 postgres
	postgresDSN        string = "host=localhost user=postgres password=quickpid dbname=quickpid port=5432 sslmode=disable"
	disableAutoMigrate bool   = false
)

func init() {
	flag.StringVar(&postgresDSN, "postgres-dsn", postgresDSN, "PostgreSQL database connection string")
	flag.BoolVar(&disableAutoMigrate, "disable-auto-migrate", disableAutoMigrate, "disable automatic database migration")
}
