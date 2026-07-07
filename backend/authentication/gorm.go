//spellchecker:words authentication
package authentication

//spellchecker:words context errors strings time github bicpid backend internal apikey gorm
import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tkw1536/bicpid/api"
	"github.com/tkw1536/bicpid/backend"
	"github.com/tkw1536/bicpid/internal/apikey"
	"gorm.io/gorm"
)

// MigrateGorm automatically migrates the gorm schema used by [NewGormBackend].
func MigrateGorm(db *gorm.DB) error {
	return db.AutoMigrate(&userRow{}, &apiKeyRow{})
}

// NewGormBackend returns [backend.AuthBackend] backed by gorm.
func NewGormBackend(db *gorm.DB) backend.AuthBackend {
	return &gormBackend{db: db, keyFormat: apikey.Default}
}

type gormBackend struct {
	db        *gorm.DB
	keyFormat apikey.Format
}

func authWithTx[V any](db *gorm.DB, fn func(*gorm.DB) (V, error)) (V, error) {
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

func isAuthUniqueConstraintError(err error) bool {
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

func (s *gormBackend) CreateUser(ctx context.Context, req api.UserCreateRequest, _ func() time.Time) (*api.UserInfo, error) {
	return authWithTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.UserInfo, error) {
		row := userRow{Username: req.Username, Superuser: req.Superuser}
		if err := tx.Create(&row).Error; err != nil {
			if isAuthUniqueConstraintError(err) {
				return nil, backend.ErrDuplicateUsername
			}
			return nil, err
		}
		info := row.toSpec()
		return &info, nil
	})
}

func (s *gormBackend) GetUser(ctx context.Context, username string) (*api.UserInfo, error) {
	return authWithTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.UserInfo, error) {
		row, err := findUser(tx, username)
		if err != nil {
			return nil, err
		}
		info := row.toSpec()
		return &info, nil
	})
}

func (s *gormBackend) ListUsers(ctx context.Context, params api.ListUsersParams) (*api.PaginatedUsersResponse, error) {
	return authWithTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.PaginatedUsersResponse, error) {
		var total int64
		if err := tx.Model(&userRow{}).Count(&total).Error; err != nil {
			return nil, err
		}

		limit := params.Limit
		offset := params.Offset
		if int64(offset) >= total {
			return &api.PaginatedUsersResponse{
				Total:  int(total),
				Offset: offset,
				Items:  []api.UserInfo{},
			}, nil
		}

		var rows []userRow
		if err := tx.Order("username ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
			return nil, err
		}
		items := make([]api.UserInfo, len(rows))
		for i := range rows {
			items[i] = rows[i].toSpec()
		}
		return &api.PaginatedUsersResponse{
			Total:  int(total),
			Offset: offset,
			Items:  items,
		}, nil
	})
}

func (s *gormBackend) DeleteUser(ctx context.Context, username string) error {
	_, err := authWithTx(s.db.WithContext(ctx), func(tx *gorm.DB) (struct{}, error) {
		var zero struct{}
		if err := ensureUserExists(tx, username); err != nil {
			return zero, err
		}
		if err := tx.Where("username = ?", username).Delete(&apiKeyRow{}).Error; err != nil {
			return zero, err
		}
		result := tx.Delete(&userRow{Username: username})
		if result.Error != nil {
			return zero, result.Error
		}
		if result.RowsAffected == 0 {
			return zero, backend.ErrUserNotFound
		}
		return zero, nil
	})
	return err
}

func (s *gormBackend) UpdateUser(ctx context.Context, username string, req api.UpdateUserRequest) (*api.UserInfo, error) {
	return authWithTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.UserInfo, error) {
		row, err := findUser(tx, username)
		if err != nil {
			return nil, err
		}
		if req.Superuser != nil {
			row.Superuser = *req.Superuser
			if err := tx.Save(&row).Error; err != nil {
				return nil, err
			}
		}
		info := row.toSpec()
		return &info, nil
	})
}

func (s *gormBackend) CreateKey(ctx context.Context, username, keyID, key string, req api.IssueKeyRequest, now func() time.Time) (*api.APIKeyInfo, error) {
	return authWithTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.APIKeyInfo, error) {
		if err := ensureUserExists(tx, username); err != nil {
			return nil, err
		}

		hashed, err := s.keyFormat.Hash(key)
		if err != nil {
			return nil, err
		}

		createdAt := now().UTC()
		row := apiKeyRow{
			Username:  username,
			ID:        keyID,
			Comment:   req.Comment,
			CreatedAt: createdAt,
			ExpiresAt: cloneStringPtr(req.ExpiresAt),
			Prefix:    hashed.Prefix,
			Digest:    hashed.Digest,
		}
		if err := tx.Create(&row).Error; err != nil {
			return nil, err
		}
		info := row.toSpec()
		return &info, nil
	})
}

func (s *gormBackend) ListKeys(ctx context.Context, username string, params api.ListKeysParams) (*api.PaginatedAPIKeysResponse, error) {
	return authWithTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.PaginatedAPIKeysResponse, error) {
		if err := ensureUserExists(tx, username); err != nil {
			return nil, err
		}

		q := tx.Model(&apiKeyRow{}).Where("username = ?", username)
		var total int64
		if err := q.Count(&total).Error; err != nil {
			return nil, err
		}

		limit := params.Limit
		offset := params.Offset
		if int64(offset) >= total {
			return &api.PaginatedAPIKeysResponse{
				Total:  int(total),
				Offset: offset,
				Items:  []api.APIKeyInfo{},
			}, nil
		}

		var rows []apiKeyRow
		if err := q.Order("id ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
			return nil, err
		}
		items := make([]api.APIKeyInfo, len(rows))
		for i := range rows {
			items[i] = rows[i].toSpec()
		}
		return &api.PaginatedAPIKeysResponse{
			Total:  int(total),
			Offset: offset,
			Items:  items,
		}, nil
	})
}

func (s *gormBackend) GetKey(ctx context.Context, username, keyID string) (*api.APIKeyInfo, error) {
	return authWithTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.APIKeyInfo, error) {
		if err := ensureUserExists(tx, username); err != nil {
			return nil, err
		}
		row, err := findKey(tx, username, keyID)
		if err != nil {
			return nil, err
		}
		info := row.toSpec()
		return &info, nil
	})
}

func (s *gormBackend) UpdateKey(ctx context.Context, username, keyID string, req api.UpdateKeyRequest, _ func() time.Time) (*api.APIKeyInfo, error) {
	return authWithTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.APIKeyInfo, error) {
		if err := ensureUserExists(tx, username); err != nil {
			return nil, err
		}
		row, err := findKey(tx, username, keyID)
		if err != nil {
			return nil, err
		}

		if req.Comment != nil {
			row.Comment = *req.Comment
		}
		if req.ExpiresAt != nil {
			row.ExpiresAt = cloneStringPtr(*req.ExpiresAt)
		}
		if err := tx.Save(&row).Error; err != nil {
			return nil, err
		}
		info := row.toSpec()
		return &info, nil
	})
}

func (s *gormBackend) RevokeKey(ctx context.Context, username, keyID string) (*api.APIKeyInfo, error) {
	return authWithTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.APIKeyInfo, error) {
		if err := ensureUserExists(tx, username); err != nil {
			return nil, err
		}
		row, err := findKey(tx, username, keyID)
		if err != nil {
			return nil, err
		}
		info := row.toSpec()
		if err := tx.Delete(&row).Error; err != nil {
			return nil, err
		}
		return &info, nil
	})
}

func (s *gormBackend) LookupUserByKey(ctx context.Context, key string) (string, error) {
	return authWithTx(s.db.WithContext(ctx), func(tx *gorm.DB) (string, error) {
		lookupPrefix, err := s.keyFormat.Prefix(key)
		if err != nil {
			return "", backend.ErrInvalidKey
		}

		var rows []apiKeyRow
		if err := tx.Where("prefix = ?", lookupPrefix).Find(&rows).Error; err != nil {
			return "", err
		}
		for _, row := range rows {
			if s.keyFormat.Verify(key, apikey.Stored{Prefix: row.Prefix, Digest: row.Digest}) {
				return row.Username, nil
			}
		}
		return "", backend.ErrInvalidKey
	})
}

var errShutdownGormAuth = errors.New("stopped waiting for shutdown to complete")

func (s *gormBackend) Shutdown(ctx context.Context) error {
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
		return fmt.Errorf("%w: %w", errShutdownGormAuth, ctx.Err())
	case err := <-result:
		return err
	}
}

func ensureUserExists(tx *gorm.DB, username string) error {
	var n int64
	if err := tx.Model(&userRow{}).Where("username = ?", username).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return backend.ErrUserNotFound
	}
	return nil
}

func findUser(tx *gorm.DB, username string) (userRow, error) {
	var row userRow
	if err := tx.First(&row, "username = ?", username).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return userRow{}, backend.ErrUserNotFound
		}
		return userRow{}, err
	}
	return row, nil
}

func findKey(tx *gorm.DB, username, keyID string) (apiKeyRow, error) {
	var row apiKeyRow
	if err := tx.First(&row, "username = ? AND id = ?", username, keyID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apiKeyRow{}, backend.ErrKeyNotFound
		}
		return apiKeyRow{}, err
	}
	return row, nil
}

type userRow struct {
	Username  string `gorm:"column:username;type:text;primaryKey"`
	Superuser bool   `gorm:"column:superuser;not null;default:false"`
}

func (userRow) TableName() string { return "auth_users" }

func (u userRow) toSpec() api.UserInfo {
	return api.UserInfo{
		Username:  u.Username,
		Superuser: u.Superuser,
	}
}

type apiKeyRow struct {
	Username string `gorm:"column:username;type:text;not null;primaryKey;index:idx_auth_api_keys_user_id,priority:1"`
	ID       string `gorm:"column:id;type:text;not null;primaryKey;index:idx_auth_api_keys_user_id,priority:2"`

	Comment   string    `gorm:"column:comment;type:text;not null"`
	CreatedAt time.Time `gorm:"column:created_at;not null"`
	ExpiresAt *string   `gorm:"column:expires_at;type:text"`

	Prefix string `gorm:"column:prefix;type:text;not null;index"`
	Digest []byte `gorm:"column:digest;type:blob;not null"`
}

func (apiKeyRow) TableName() string { return "auth_api_keys" }

func (k apiKeyRow) toSpec() api.APIKeyInfo {
	return api.APIKeyInfo{
		ID:        k.ID,
		Comment:   k.Comment,
		CreatedAt: k.CreatedAt.UTC().Format(time.RFC3339),
		ExpiresAt: cloneStringPtr(k.ExpiresAt),
	}
}
