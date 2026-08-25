//spellchecker:words gorm
package gorm

//spellchecker:words context github quickpid backend gorm
import (
	"context"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
	"gorm.io/gorm"
)

type userMountGrantRow struct {
	Username string `gorm:"column:username;type:text;not null;primaryKey"`
	Pattern  string `gorm:"column:pattern;type:text;not null;primaryKey"`
}

func (userMountGrantRow) TableName() string { return "user_mount_grants" }

func (s *GormBackend) ListUserMountGrants(ctx context.Context, username api.ValidUsername, params api.ListUserMountGrantsParams) (*api.PaginatedMountGrantsResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.PaginatedMountGrantsResponse, error) {
		if err := ensureUserExists(tx, username); err != nil {
			return nil, err
		}

		q := tx.Model(&userMountGrantRow{}).Where("username = ?", username.String())

		var total int64
		if err := q.Count(&total).Error; err != nil {
			return nil, err
		}

		if params.Offset >= int(total) {
			return &api.PaginatedMountGrantsResponse{
				Total:  int(total),
				Offset: params.Offset,
				Items:  []api.MountGrant{},
			}, nil
		}

		var rows []userMountGrantRow
		if err := q.Order("pattern asc").Offset(params.Offset).Limit(params.Limit).Find(&rows).Error; err != nil {
			return nil, err
		}

		items := make([]api.MountGrant, len(rows))
		for i := range rows {
			items[i] = api.MountGrant{Pattern: rows[i].Pattern}
		}
		return &api.PaginatedMountGrantsResponse{
			Total:  int(total),
			Offset: params.Offset,
			Items:  items,
		}, nil
	})
}

func (s *GormBackend) SetMountGrant(ctx context.Context, username api.ValidUsername, req api.ValidMountGrantRequest) error {
	_, err := withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (struct{}, error) {
		var zero struct{}
		if err := ensureUserExists(tx, username); err != nil {
			return zero, err
		}

		row := userMountGrantRow{
			Username: username.String(),
			Pattern:  req.String(),
		}
		if err := tx.Save(&row).Error; err != nil {
			return zero, err
		}
		return zero, nil
	})
	return err
}

func (s *GormBackend) DeleteMountGrant(ctx context.Context, username api.ValidUsername, req api.ValidMountGrantRequest) error {
	_, err := withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (struct{}, error) {
		var zero struct{}
		if err := ensureUserExists(tx, username); err != nil {
			return zero, err
		}

		result := tx.Where("username = ? AND pattern = ?", username.String(), req.String()).Delete(&userMountGrantRow{})
		if result.Error != nil {
			return zero, result.Error
		}
		if result.RowsAffected == 0 {
			return zero, backend.ErrMountGrantNotFound
		}
		return zero, nil
	})
	return err
}
