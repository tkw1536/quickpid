// Package gorm provides a GORM-backed [backend.Store].
//
//spellchecker:words gorm
package gorm

//spellchecker:words context errors strings github bicpid backend internal apikey gorm
import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/tkw1536/bicpid/backend"
	"github.com/tkw1536/bicpid/internal/apikey"
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
	return db.AutoMigrate(
		&namespaceRow{},
		&resourceRow{},
		&userRow{},
		&apiKeyRow{},
		&namespacePermissionRow{},
	)
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
		return zero, err
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

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint failed") ||
		strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "duplicated") ||
		strings.Contains(msg, "unique constraint")
}
