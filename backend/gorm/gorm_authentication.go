//spellchecker:words gorm
package gorm

//spellchecker:words context time github bicpid backend internal apikey gorm
import (
	"context"
	"time"

	"github.com/tkw1536/bicpid/api"
	"github.com/tkw1536/bicpid/backend"
	"github.com/tkw1536/bicpid/internal/apikey"
	"gorm.io/gorm"
)

func (s *Store) CreateUser(ctx context.Context, req api.UserCreateRequest, _ func() time.Time) (*api.UserInfo, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.UserInfo, error) {
		row := userRow{Username: req.Username, Superuser: req.Superuser}
		if err := tx.Create(&row).Error; err != nil {
			if isUniqueConstraintError(err) {
				return nil, backend.ErrDuplicateUsername
			}
			return nil, err
		}
		info := row.toSpec()
		return &info, nil
	})
}

func (s *Store) GetUser(ctx context.Context, username string) (*api.UserInfo, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.UserInfo, error) {
		row, err := findUser(tx, username)
		if err != nil {
			return nil, err
		}
		info := row.toSpec()
		return &info, nil
	})
}

func (s *Store) ListUsers(ctx context.Context, params api.ListUsersParams) (*api.PaginatedUsersResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.PaginatedUsersResponse, error) {
		q := tx.Model(&userRow{})
		if params.Superuser != nil {
			q = q.Where("superuser = ?", *params.Superuser)
		}
		var total int64
		if err := q.Count(&total).Error; err != nil {
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
		if err := q.Order("username ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
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

func (s *Store) AutocompleteUsers(ctx context.Context, query string, limit int) ([]string, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) ([]string, error) {
		var rows []userRow
		if err := tx.Where("username LIKE ?", query+"%").Order("username ASC").Limit(limit).Find(&rows).Error; err != nil {
			return nil, err
		}
		usernames := make([]string, len(rows))
		for i := range rows {
			usernames[i] = rows[i].Username
		}
		return usernames, nil
	})
}

func (s *Store) DeleteUser(ctx context.Context, username string) error {
	_, err := withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (struct{}, error) {
		var zero struct{}
		if err := ensureUserExists(tx, username); err != nil {
			return zero, err
		}
		if err := tx.Where("username = ?", username).Delete(&apiKeyRow{}).Error; err != nil {
			return zero, err
		}
		if err := tx.Where("username = ?", username).Delete(&namespacePermissionRow{}).Error; err != nil {
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

func (s *Store) UpdateUser(ctx context.Context, username string, req api.UserUpdateRequest) (*api.UserInfo, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.UserInfo, error) {
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

func (s *Store) CreateKey(ctx context.Context, format apikey.Format, username, keyID string, key string, req api.KeyIssueRequest, now func() time.Time) (*api.APIKeyInfo, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.APIKeyInfo, error) {
		if err := ensureUserExists(tx, username); err != nil {
			return nil, err
		}

		hashed, err := format.Hash(key)
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
			if isUniqueConstraintError(err) {
				return nil, backend.ErrKeyCollision
			}
			return nil, err
		}
		info := row.toSpec()
		return &info, nil
	})
}

func (s *Store) ListKeys(ctx context.Context, _ apikey.Format, username string, params api.ListKeysParams) (*api.PaginatedAPIKeysResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.PaginatedAPIKeysResponse, error) {
		if err := ensureUserExists(tx, username); err != nil {
			return nil, err
		}

		q := tx.Model(&apiKeyRow{}).Where("username = ? AND revoked = ?", username, false)
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

func (s *Store) GetKey(ctx context.Context, _ apikey.Format, username, keyID string) (*api.APIKeyInfo, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.APIKeyInfo, error) {
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

func (s *Store) UpdateKey(ctx context.Context, _ apikey.Format, username, keyID string, req api.KeyUpdateRequest, _ func() time.Time) (*api.APIKeyInfo, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.APIKeyInfo, error) {
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

func (s *Store) RevokeKey(ctx context.Context, _ apikey.Format, username, keyID string) (*api.APIKeyInfo, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.APIKeyInfo, error) {
		if err := ensureUserExists(tx, username); err != nil {
			return nil, err
		}
		row, err := findKeyIncludingRevoked(tx, username, keyID)
		if err != nil {
			return nil, err
		}
		info := row.toSpec()
		if row.Revoked {
			return &info, nil
		}
		row.Revoked = true
		if err := tx.Save(&row).Error; err != nil {
			return nil, err
		}
		return &info, nil
	})
}

func (s *Store) LookupUserByKey(ctx context.Context, format apikey.Format, key string) (string, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (string, error) {
		lookupPrefix, err := format.Prefix(key)
		if err != nil {
			return "", backend.ErrInvalidKey
		}

		var rows []apiKeyRow
		if err := tx.Where("prefix = ?", lookupPrefix).Find(&rows).Error; err != nil {
			return "", err
		}
		for _, row := range rows {
			if row.Revoked {
				continue
			}
			if format.Verify(key, apikey.Stored{Prefix: row.Prefix, Digest: row.Digest}) {
				return row.Username, nil
			}
		}
		return "", backend.ErrInvalidKey
	})
}
