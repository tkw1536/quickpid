//spellchecker:words gorm
package gorm_test

//spellchecker:words testing github glebarez sqlite quickpid backend gorm gormstore internal pidtest servertest logger
import (
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/tkw1536/quickpid/backend"
	gormstore "github.com/tkw1536/quickpid/backend/gorm"
	servertest "github.com/tkw1536/quickpid/internal/pidtest"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

//spellchecker:words nolint paralleltest

// This file contains tests for the postgres backend.
// These require a running postgres database, which needs to be set up via the
// TEST_POSTGRES_DSN environment variable.
//
// The database schema is automatically reset between tests,
// and any state in the database is lost.

var postgresDSN string = os.Getenv("TEST_POSTGRES_DSN")

func newPostgresStore(t *testing.T, l *slog.Logger) backend.Store {
	t.Helper()

	db, err := gorm.Open(postgres.Open(postgresDSN), &gorm.Config{
		Logger: gormLogger.NewSlogLogger(l, gormLogger.Config{
			LogLevel:                  gormLogger.Info,
			IgnoreRecordNotFoundError: true,
		}),
	})

	if err != nil {
		t.Fatal(err)
	}
	if err := gormstore.Migrate(db); err != nil {
		t.Fatal(err)
	}
	return gormstore.NewStore(db, 0)
}

// resetPostgresStore resets the postgres store by dropping all tables.
func resetPostgresStore(t *testing.T) error {
	t.Helper()

	db, err := gorm.Open(postgres.Open(postgresDSN), &gorm.Config{
		Logger: gormLogger.Discard,
	})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			t.Fatalf("failed to close database connection: %s", err)
		}
	}()

	if err := db.Exec(`DROP SCHEMA IF EXISTS public CASCADE`).Error; err != nil {
		return fmt.Errorf("failed to drop schema: %w", err)
	}
	if err := db.Exec(`CREATE SCHEMA public`).Error; err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}
	return nil
}

//nolint:paralleltest // we need a single running postgres database for these tests
func TestStore_postgres(t *testing.T) {
	if postgresDSN == "" {
		t.Skip("Set TEST_POSTGRES_DSN to test postgres functionality")
	}

	servertest.RunStoreTests(t, newPostgresStore, resetPostgresStore)
}

//nolint:paralleltest // we need a single running postgres database for these tests
func TestStore_Flows_postgres(t *testing.T) {
	if postgresDSN == "" {
		t.Skip("Set TEST_POSTGRES_DSN to test postgres functionality")
	}

	servertest.RunFlowTests(t, newPostgresStore, resetPostgresStore)
}
