package gorm

//spellchecker:words errors strings gorm github bicpid backend
import (
	"errors"
	"strings"

	"github.com/tkw1536/bicpid/backend"
	"gorm.io/gorm"
)

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
