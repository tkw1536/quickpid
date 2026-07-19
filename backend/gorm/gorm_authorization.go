//spellchecker:words gorm
package gorm

//spellchecker:words context errors github quickpid backend gorm
import (
	"context"
	"errors"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
	"gorm.io/gorm"
)

func (s *Store) GetNamespacePermission(ctx context.Context, namespace *api.ValidNamespaceID, username *api.ValidUsername) (api.PermissionLevel, error) {
	level, err := withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (api.PermissionLevel, error) {
		if err := ensureNamespaceExists(tx, namespace); err != nil {
			return "", err
		}

		var row namespacePermissionRow
		if err := tx.First(&row, "namespace = ? AND username = ?", namespace.String(), username.String()).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return api.PermissionLevelNone, nil
			}
			return "", err
		}
		return api.PermissionLevel(row.Level), nil
	})
	return level, err
}

func (s *Store) SetNamespacePermission(ctx context.Context, namespace *api.ValidNamespaceID, username *api.ValidUsername, level api.PermissionLevel) error {
	if level.Validate() != nil {
		return backend.ErrInvalidPermissionLevel
	}

	_, err := withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (struct{}, error) {
		var zero struct{}
		if err := ensureNamespaceExists(tx, namespace); err != nil {
			return zero, err
		}

		if level == api.PermissionLevelNone {
			result := tx.Where("namespace = ? AND username = ?", namespace.String(), username.String()).Delete(&namespacePermissionRow{})
			if result.Error != nil {
				return zero, result.Error
			}
			return zero, nil
		}

		row := namespacePermissionRow{
			Namespace: namespace.String(),
			Username:  username.String(),
			Level:     string(level),
		}
		if err := tx.Save(&row).Error; err != nil {
			return zero, err
		}
		return zero, nil
	})
	return err
}

func (s *Store) DeleteNamespacePermission(ctx context.Context, namespace *api.ValidNamespaceID, username *api.ValidUsername) error {
	_, err := withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (struct{}, error) {
		var zero struct{}
		result := tx.Where("namespace = ? AND username = ?", namespace.String(), username.String()).Delete(&namespacePermissionRow{})
		if result.Error != nil {
			return zero, result.Error
		}
		if result.RowsAffected == 0 {
			return zero, backend.ErrPermissionNotFound
		}
		return zero, nil
	})
	return err
}

func (s *Store) ListNamespacePermissions(ctx context.Context, namespace *api.ValidNamespaceID, params api.ListNamespacePermissionsParams) (*api.PaginatedNamespacePermissionsResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.PaginatedNamespacePermissionsResponse, error) {
		if err := ensureNamespaceExists(tx, namespace); err != nil {
			return nil, err
		}

		q := tx.Model(&namespacePermissionRow{}).Where("namespace = ?", namespace.String())
		var total int64
		if err := q.Count(&total).Error; err != nil {
			return nil, err
		}

		limit := params.Limit
		offset := params.Offset
		if limit == 0 || int64(offset) >= total {
			return &api.PaginatedNamespacePermissionsResponse{
				Total:  int(total),
				Offset: offset,
				Items:  []api.NamespacePermission{},
			}, nil
		}

		var rows []namespacePermissionRow
		if err := q.Order("username ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
			return nil, err
		}
		items := make([]api.NamespacePermission, len(rows))
		for i := range rows {
			items[i] = rows[i].toSpec()
		}
		return &api.PaginatedNamespacePermissionsResponse{
			Total:  int(total),
			Offset: offset,
			Items:  items,
		}, nil
	})
}
