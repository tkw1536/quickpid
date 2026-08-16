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

type namespaceRoleRow struct {
	Namespace string `gorm:"column:namespace;type:text;not null;primaryKey"`
	Username  string `gorm:"column:username;type:text;not null;primaryKey"`
	Role      string `gorm:"column:role;type:text;not null"`
}

func (namespaceRoleRow) TableName() string { return "authz_namespace_roles" }

func (p namespaceRoleRow) toSpec() api.NamespaceRole {
	return api.NamespaceRole{
		Username: p.Username,
		Role:     api.Role(p.Role),
	}
}

func (p namespaceRoleRow) toUserRole() api.UserRole {
	return api.UserRole{
		Namespace: p.Namespace,
		Role:      api.Role(p.Role),
	}
}

func (s *GormBackend) GetNamespaceRole(ctx context.Context, namespace api.ValidNamespaceID, username api.ValidUsername) (api.Role, error) {
	role, err := withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (api.Role, error) {
		if err := ensureNamespaceExists(tx, namespace); err != nil {
			return "", err
		}

		var row namespaceRoleRow
		if err := tx.First(&row, "namespace = ? AND username = ?", namespace.String(), username.String()).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return api.RoleNone, nil
			}
			return "", err
		}
		return api.Role(row.Role), nil
	})
	return role, err
}

func (s *GormBackend) SetNamespaceRole(ctx context.Context, namespace api.ValidNamespaceID, username api.ValidUsername, role api.Role) error {
	if role.Validate() != nil {
		return backend.ErrInvalidRole
	}

	_, err := withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (struct{}, error) {
		var zero struct{}
		if err := ensureNamespaceExists(tx, namespace); err != nil {
			return zero, err
		}

		if role == api.RoleNone {
			result := tx.Where("namespace = ? AND username = ?", namespace.String(), username.String()).Delete(&namespaceRoleRow{})
			if result.Error != nil {
				return zero, result.Error
			}
			return zero, nil
		}

		row := namespaceRoleRow{
			Namespace: namespace.String(),
			Username:  username.String(),
			Role:      string(role),
		}
		if err := tx.Save(&row).Error; err != nil {
			return zero, err
		}
		return zero, nil
	})
	return err
}

func (s *GormBackend) DeleteNamespaceRole(ctx context.Context, namespace api.ValidNamespaceID, username api.ValidUsername) error {
	_, err := withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (struct{}, error) {
		var zero struct{}
		result := tx.Where("namespace = ? AND username = ?", namespace.String(), username.String()).Delete(&namespaceRoleRow{})
		if result.Error != nil {
			return zero, result.Error
		}
		if result.RowsAffected == 0 {
			return zero, backend.ErrRoleNotFound
		}
		return zero, nil
	})
	return err
}

func (s *GormBackend) ListNamespaceRoles(ctx context.Context, namespace api.ValidNamespaceID, params api.ListNamespaceRolesParams) (*api.PaginatedNamespaceRolesResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.PaginatedNamespaceRolesResponse, error) {
		if err := ensureNamespaceExists(tx, namespace); err != nil {
			return nil, err
		}

		q := tx.Model(&namespaceRoleRow{}).Where("namespace = ?", namespace.String())
		var total int64
		if err := q.Count(&total).Error; err != nil {
			return nil, err
		}

		limit := params.Limit
		offset := params.Offset
		if limit == 0 || int64(offset) >= total {
			return &api.PaginatedNamespaceRolesResponse{
				Total:  int(total),
				Offset: offset,
				Items:  []api.NamespaceRole{},
			}, nil
		}

		var rows []namespaceRoleRow
		if err := q.Order("username ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
			return nil, err
		}
		items := make([]api.NamespaceRole, len(rows))
		for i := range rows {
			items[i] = rows[i].toSpec()
		}
		return &api.PaginatedNamespaceRolesResponse{
			Total:  int(total),
			Offset: offset,
			Items:  items,
		}, nil
	})
}

func (s *GormBackend) ListUserRoles(ctx context.Context, username api.ValidUsername, params api.ListUserRolesParams) (*api.PaginatedUserRolesResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.PaginatedUserRolesResponse, error) {
		if err := ensureUserExists(tx, username); err != nil {
			return nil, err
		}

		q := tx.Model(&namespaceRoleRow{}).Where("username = ?", username.String())
		var total int64
		if err := q.Count(&total).Error; err != nil {
			return nil, err
		}

		limit := params.Limit
		offset := params.Offset
		if limit == 0 || int64(offset) >= total {
			return &api.PaginatedUserRolesResponse{
				Total:  int(total),
				Offset: offset,
				Items:  []api.UserRole{},
			}, nil
		}

		var rows []namespaceRoleRow
		if err := q.Order("namespace ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
			return nil, err
		}
		items := make([]api.UserRole, len(rows))
		for i := range rows {
			items[i] = rows[i].toUserRole()
		}
		return &api.PaginatedUserRolesResponse{
			Total:  int(total),
			Offset: offset,
			Items:  items,
		}, nil
	})
}
