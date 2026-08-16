//spellchecker:words gorm
package gorm

//spellchecker:words context errors time github quickpid backend gorm
import (
	"context"
	"errors"
	"time"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
	"gorm.io/gorm"
)

type userRow struct {
	Username     string `gorm:"column:username;type:text;primaryKey"`
	Superuser    bool   `gorm:"column:superuser;not null;default:false"`
	PasswordHash []byte `gorm:"column:password_hash"`
}

func (userRow) TableName() string { return "auth_users" }

func (u userRow) toSpec() api.UserInfo {
	return api.UserInfo{
		Username:  u.Username,
		Superuser: u.Superuser,
		Password:  len(u.PasswordHash) > 0,
	}
}

func ensureUserExists(tx *gorm.DB, user api.ValidUsername) error {
	var n int64
	if err := tx.Model(&userRow{}).Where("username = ?", user.String()).Count(&n).Limit(1).Error; err != nil {
		return err
	}
	if n == 0 {
		return errUserNotFound
	}
	return nil
}

func findUser(tx *gorm.DB, username api.ValidUsername) (userRow, error) {
	var row userRow
	if err := tx.First(&row, "username = ?", username.String()).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return userRow{}, errUserNotFound
		}
		return userRow{}, err
	}
	return row, nil
}

func (s *Store) CreateUser(ctx context.Context, req api.ValidUserCreateRequest, _ func() time.Time) (*api.UserInfo, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.UserInfo, error) {
		row := userRow{Username: req.Username.String(), Superuser: req.Superuser}
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

func (s *Store) GetUser(ctx context.Context, username api.ValidUsername) (*api.UserInfo, error) {
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
		if params.Password != nil {
			if *params.Password {
				q = q.Where("password_hash IS NOT NULL AND length(password_hash) > 0")
			} else {
				q = q.Where("password_hash IS NULL OR length(password_hash) = 0")
			}
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

func (s *Store) AutocompleteUsers(ctx context.Context, query api.ValidAutocompleteQuery, limit int) ([]string, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) ([]string, error) {
		var rows []userRow
		if err := tx.Where("username LIKE ?", query.String()+"%").Order("username ASC").Limit(limit).Find(&rows).Error; err != nil {
			return nil, err
		}
		usernames := make([]string, len(rows))
		for i := range rows {
			usernames[i] = rows[i].Username
		}
		return usernames, nil
	})
}

func (s *Store) DeleteUser(ctx context.Context, username api.ValidUsername) error {
	_, err := withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (struct{}, error) {
		var zero struct{}
		if err := ensureUserExists(tx, username); err != nil {
			return zero, err
		}
		if err := tx.Where("username = ?", username.String()).Delete(&apiKeyRow{}).Error; err != nil {
			return zero, err
		}
		if err := tx.Where("username = ?", username.String()).Delete(&namespaceRoleRow{}).Error; err != nil {
			return zero, err
		}
		result := tx.Delete(&userRow{Username: username.String()})
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

func (s *Store) UpdateUser(ctx context.Context, username api.ValidUsername, req api.UserUpdateRequest) (*api.UserInfo, error) {
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
