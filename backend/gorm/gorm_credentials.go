//spellchecker:words gorm
package gorm

//spellchecker:words context time github quickpid backend internal apikey password gorm
import (
	"context"
	"fmt"
	"time"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/internal/apikey"
	"github.com/tkw1536/quickpid/internal/password"
	"gorm.io/gorm"
)

type apiKeyRow struct {
	Username string `gorm:"column:username;type:text;not null;primaryKey;index:idx_auth_api_keys_user_id,priority:1"`
	ID       string `gorm:"column:id;type:text;not null;primaryKey;index:idx_auth_api_keys_user_id,priority:2"`

	Comment   string     `gorm:"column:comment;type:text;not null"`
	CreatedAt time.Time  `gorm:"column:created_at;not null"`
	ExpiresAt *time.Time `gorm:"column:expires_at"`

	Prefix string `gorm:"column:prefix;type:text;not null;index"`
	Digest []byte `gorm:"column:digest;not null"`
}

func (apiKeyRow) TableName() string { return "auth_api_keys" }

func (k apiKeyRow) toSpec() api.APIKeyInfo {
	expiresAt := k.ExpiresAt
	if expiresAt != nil {
		utc := expiresAt.UTC()
		expiresAt = &utc
	}
	return api.APIKeyInfo{
		ID:        k.ID,
		Comment:   k.Comment,
		CreatedAt: k.CreatedAt.UTC(),
		ExpiresAt: expiresAt,
	}
}

func (s *Store) SetPassword(ctx context.Context, username api.ValidUsername, newPassword *api.ValidPassword) (bool, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (bool, error) {
		row, err := findUser(tx, username)
		if err != nil {
			return false, err
		}
		if newPassword == nil {
			row.PasswordHash = nil
		} else {
			hash, hashErr := password.Hash(newPassword.String())
			if hashErr != nil {
				return false, fmt.Errorf("failed to hash password: %w", hashErr)
			}
			row.PasswordHash = hash
		}
		if err := tx.Save(&row).Error; err != nil {
			return false, err
		}
		return len(row.PasswordHash) > 0, nil
	})
}

func (s *Store) CheckPassword(ctx context.Context, username api.ValidUsername, candidate api.ValidPassword) (bool, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (bool, error) {
		row, err := findUser(tx, username)
		if err != nil {
			return false, err
		}
		if len(row.PasswordHash) == 0 {
			return false, nil
		}
		return password.Verify(candidate.String(), row.PasswordHash), nil
	})
}

func (s *Store) CreateKey(ctx context.Context, format apikey.Format, username api.ValidUsername, keyID string, key string, req api.KeyIssueRequest, now func() time.Time) (*api.APIKeyInfo, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.APIKeyInfo, error) {
		if err := ensureUserExists(tx, username); err != nil {
			return nil, err
		}

		hashed, err := format.Hash(key)
		if err != nil {
			return nil, fmt.Errorf("failed to hash key: %w", err)
		}

		createdAt := now().UTC()
		row := apiKeyRow{
			Username:  username.String(),
			ID:        keyID,
			Comment:   req.Comment,
			CreatedAt: createdAt,
			ExpiresAt: req.ExpiresAt,
			Prefix:    hashed.Prefix,
			Digest:    hashed.Digest,
		}
		if err := tx.Create(&row).Error; err != nil {
			if isUniqueConstraintError(err) {
				return nil, backend.ErrKeyCollision
			}
			return nil, err
		}
		info := row.toSpec()
		return &info, nil
	})
}

func (s *Store) ListKeys(ctx context.Context, _ apikey.Format, username api.ValidUsername, params api.ListKeysParams) (*api.PaginatedAPIKeysResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.PaginatedAPIKeysResponse, error) {
		if err := ensureUserExists(tx, username); err != nil {
			return nil, err
		}

		q := tx.Model(&apiKeyRow{}).Where("username = ?", username.String())
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

func (s *Store) RevokeKey(ctx context.Context, _ apikey.Format, username api.ValidUsername, keyID string) error {
	_, err := withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (struct{}, error) {
		var zero struct{}
		if err := ensureUserExists(tx, username); err != nil {
			return zero, err
		}
		deleteQuery := tx.Where("username = ? AND id = ?", username.String(), keyID).Delete(&apiKeyRow{})
		if err := deleteQuery.Error; err != nil {
			return zero, err
		}
		if deleteQuery.RowsAffected == 0 {
			return zero, backend.ErrKeyNotFound
		}
		return zero, nil
	})
	return err
}

func (s *Store) LookupUserByKey(ctx context.Context, format apikey.Format, key string) (string, *api.APIKeyInfo, error) {
	return withTx2(s.db.WithContext(ctx), func(tx *gorm.DB) (string, *api.APIKeyInfo, error) {
		lookupPrefix, err := format.Prefix(key)
		if err != nil {
			return "", nil, fmt.Errorf("%w: invalid key prefix", backend.ErrInvalidKey)
		}

		var rows []apiKeyRow
		if err := tx.Where("prefix = ?", lookupPrefix).Find(&rows).Error; err != nil {
			return "", nil, err
		}
		for _, row := range rows {
			if format.Verify(key, apikey.Stored{Prefix: row.Prefix, Digest: row.Digest}) {
				return row.Username, new(row.toSpec()), nil
			}
		}
		return "", nil, fmt.Errorf("%w: no valid key found", backend.ErrInvalidKey)
	})
}
