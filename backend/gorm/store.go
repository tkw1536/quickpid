// Package gorm provides a GORM-backed [backend.Store].
package gorm

//spellchecker:words gorm github bicpid backend internal apikey
import (
	"github.com/tkw1536/bicpid/internal/apikey"
	"gorm.io/gorm"
)

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
