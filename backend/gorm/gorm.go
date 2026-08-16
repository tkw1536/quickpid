// Package gorm provides a GORM-backed [backend.Backend].
//
//spellchecker:words gorm
package gorm

//spellchecker:words context errors strings github quickpid backend internal apikey gorm clause
import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/internal/apikey"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// NewGormBackend returns a GORM-backed backend.
//
// batchSize is used during batch resource creation. If <= 0, [DefaultBatchSize] is used.
func NewGormBackend(db *gorm.DB, batchSize int) *GormBackend {
	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}
	return &GormBackend{
		db:        db,
		batchSize: batchSize,
		keyFormat: apikey.Default,
	}
}

// GormBackend implements [backend.Backend] using GORM.
type GormBackend struct {
	db        *gorm.DB
	batchSize int
	keyFormat apikey.Format
}

var errShutdownGormBackend = errors.New("stopped waiting for shutdown to complete")

func (s *GormBackend) Shutdown(ctx context.Context) error {
	result := make(chan error, 1)
	go func() {
		defer close(result)

		sqlDB, err := s.db.DB()
		if err != nil {
			result <- fmt.Errorf("failed to get sql.DB: %w", err)
			return
		}
		if err := sqlDB.Close(); err != nil {
			result <- fmt.Errorf("failed to close sql.DB: %w", err)
			return
		}
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("%w: %w", errShutdownGormBackend, ctx.Err())
	case err := <-result:
		return err
	}
}

// DefaultBatchSize is the default batch size for batch create operations.
const DefaultBatchSize = 100

// Migrate migrates all tables used by [NewGormBackend].
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&userRow{},
		&userPasswordRow{},
		&apiKeyRow{},
		&apiKeyUserScopeRow{},
		&apiKeyNamespaceScopeRow{},
		&namespaceRow{},
		&namespaceTagRow{},
		&namespaceRoleRow{},
		&mountRow{},
		&resourceRow{},
		&resourceTagRow{},
	); err != nil {
		return fmt.Errorf("failed to auto-migrate database: %w", err)
	}
	return nil
}

// SQLiteForeignKeysEnabled reports whether foreign key enforcement is enabled
// on the given SQLite database connection.
func SQLiteForeignKeysEnabled(db *gorm.DB) (bool, error) {
	var foreignKeys int
	if err := db.Raw("PRAGMA foreign_keys").Scan(&foreignKeys).Error; err != nil {
		return false, fmt.Errorf("failed to run sqlite foreign keys pragma query: %w", err)
	}
	return foreignKeys != 0, nil
}

var (
	errNamespaceNotFound = backend.ErrNamespaceNotFound
	errUserNotFound      = backend.ErrUserNotFound
)

func cascadingDelete(tx *gorm.DB, value any) error {
	return tx.Select(clause.Associations).Delete(value).Error
}

func withTx[V any](db *gorm.DB, fn func(*gorm.DB) (V, error)) (V, error) {
	v, _, err := withTx2(db, func(d *gorm.DB) (V, struct{}, error) {
		v, err := fn(d)
		return v, struct{}{}, err
	})
	return v, err
}

func withTx2[V, W any](db *gorm.DB, fn func(*gorm.DB) (V, W, error)) (V, W, error) {
	var out V
	var out2 W
	if err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		out, out2, err = fn(tx)
		return err
	}); err != nil {
		var zero V
		var zero2 W
		return zero, zero2, fmt.Errorf("database transaction failed: %w", err)
	}
	return out, out2, nil
}

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}

	// HACK HACK HACK: We shouldn't be checking the string error message here.
	// But there isn't a nice way to do this, as some gorm drivers don't handle this properly.
	// TODO: Investigate if we can handle this better for our own drivers.
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint failed") ||
		strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "duplicated") ||
		strings.Contains(msg, "unique constraint")
}
