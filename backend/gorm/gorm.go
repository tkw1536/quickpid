// Package gorm provides a GORM-backed [backend.Store].
//
//spellchecker:words gorm
package gorm

//spellchecker:words context errors strings github quickpid backend internal apikey gorm
import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/internal/apikey"
	"gorm.io/gorm"
)

// NewStore returns a GORM-backed store.
//
// batchSize is used during batch resource creation. If <= 0, [DefaultBatchSize] is used.
func NewStore(db *gorm.DB, batchSize int) *Store {
	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}
	return &Store{
		db:        db,
		batchSize: batchSize,
		keyFormat: apikey.Default,
	}
}

// Store implements [backend.Store] using GORM.
type Store struct {
	db        *gorm.DB
	batchSize int
	keyFormat apikey.Format
}

var errShutdownGormStore = errors.New("stopped waiting for shutdown to complete")

func (s *Store) Shutdown(ctx context.Context) error {
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
		return fmt.Errorf("%w: %w", errShutdownGormStore, ctx.Err())
	case err := <-result:
		return err
	}
}

// DefaultBatchSize is the default batch size for batch create operations.
const DefaultBatchSize = 100

// Migrate migrates all tables used by [NewStore].
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&namespaceRow{},
		&resourceRow{},
		&userRow{},
		&apiKeyRow{},
		&namespacePermissionRow{},
	); err != nil {
		return fmt.Errorf("gorm.DB.AutoMigrate: %w", err)
	}
	return nil
}

var (
	errNamespaceNotFound = backend.ErrNamespaceNotFound
	errUserNotFound      = backend.ErrUserNotFound
	errKeyNotFound       = backend.ErrKeyNotFound
)

func withTx[V any](db *gorm.DB, fn func(*gorm.DB) (V, error)) (V, error) {
	var out V
	if err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		out, err = fn(tx)
		return err
	}); err != nil {
		var zero V
		return zero, fmt.Errorf("gorm.DB.Transaction: %w", err)
	}
	return out, nil
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
