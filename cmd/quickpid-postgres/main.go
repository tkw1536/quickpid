//spellchecker:words main
package main

//spellchecker:words flag slog github bicpid backend gorm gormstore driver postgres
import (
	"cmp"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/tkw1536/bicpid/backend"
	gormstore "github.com/tkw1536/bicpid/backend/gorm"
	"github.com/tkw1536/bicpid/cmd"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	var db *gorm.DB
	cmd.Main("quickpid-postgres",
		func(logger *slog.Logger) error {
			logger.Info("opening database", "dsn", postgresDSN)
			var err error
			db, err = gorm.Open(postgres.Open(postgresDSN), &gorm.Config{})
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
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
	// spin up a temporary local postgres with something like:
	// docker run --rm -e POSTGRES_PASSWORD=quickpid -e POSTGRES_DB=quickpid -p 5432:5432 postgres
	postgresDSN        string = cmp.Or(os.Getenv("DSN"), "host=localhost user=postgres password=quickpid dbname=quickpid port=5432 sslmode=disable")
	disableAutoMigrate bool   = false
)

func init() {
	flag.StringVar(&postgresDSN, "dsn", postgresDSN, "PostgreSQL database connection string (can also be set via DSN environment variable)")
	flag.BoolVar(&disableAutoMigrate, "disable-auto-migrate", disableAutoMigrate, "disable automatic database migration")
}
