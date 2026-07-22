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

func (s *Store) GetMount(ctx context.Context, baseURI api.ValidBaseURI) (string, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (string, error) {
		var row mountRow
		if err := tx.First(&row, "base_uri = ?", baseURI.String()).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "", backend.ErrMountNotFound
			}
			return "", err
		}
		return row.NamespaceID, nil
	})
}

func (s *Store) SetMount(ctx context.Context, baseURI api.ValidBaseURI, namespace api.ValidNamespaceID) error {
	_, err := withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (struct{}, error) {
		var zero struct{}
		if err := ensureNamespaceExists(tx, namespace); err != nil {
			return zero, err
		}
		row := mountRow{
			BaseURI:     baseURI.String(),
			NamespaceID: namespace.String(),
		}
		if err := tx.Save(&row).Error; err != nil {
			return zero, err
		}
		return zero, nil
	})
	return err
}

func (s *Store) DeleteMount(ctx context.Context, baseURI api.ValidBaseURI) error {
	_, err := withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (struct{}, error) {
		var zero struct{}
		result := tx.Where("base_uri = ?", baseURI.String()).Delete(&mountRow{})
		if result.Error != nil {
			return zero, result.Error
		}
		if result.RowsAffected == 0 {
			return zero, backend.ErrMountNotFound
		}
		return zero, nil
	})
	return err
}

func (s *Store) ListMounts(ctx context.Context, params api.ListMountsParams) (*api.PaginatedMountsResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.PaginatedMountsResponse, error) {
		q := tx.Model(&mountRow{})
		var total int64
		if err := q.Count(&total).Error; err != nil {
			return nil, err
		}

		limit := params.Limit
		offset := params.Offset
		if limit == 0 || int64(offset) >= total {
			return &api.PaginatedMountsResponse{
				Total:  int(total),
				Offset: offset,
				Items:  []api.MountResponse{},
			}, nil
		}

		var rows []mountRow
		if err := q.Order("namespace_id ASC, base_uri ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
			return nil, err
		}
		items := make([]api.MountResponse, len(rows))
		for i := range rows {
			items[i] = api.MountResponse{
				BaseURI:   rows[i].BaseURI,
				Namespace: rows[i].NamespaceID,
			}
		}
		return &api.PaginatedMountsResponse{
			Total:  int(total),
			Offset: offset,
			Items:  items,
		}, nil
	})
}

func (s *Store) ListNamespaceMounts(ctx context.Context, namespace api.ValidNamespaceID, params api.ListNamespaceMountsParams) (*api.PaginatedBaseURIResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.PaginatedBaseURIResponse, error) {
		if err := ensureNamespaceExists(tx, namespace); err != nil {
			return nil, err
		}

		q := tx.Model(&mountRow{}).Where("namespace_id = ?", namespace.String())
		var total int64
		if err := q.Count(&total).Error; err != nil {
			return nil, err
		}

		limit := params.Limit
		offset := params.Offset
		if limit == 0 || int64(offset) >= total {
			return &api.PaginatedBaseURIResponse{
				Total:  int(total),
				Offset: offset,
				Items:  []string{},
			}, nil
		}

		var rows []mountRow
		if err := q.Order("base_uri ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
			return nil, err
		}
		items := make([]string, len(rows))
		for i := range rows {
			items[i] = rows[i].BaseURI
		}
		return &api.PaginatedBaseURIResponse{
			Total:  int(total),
			Offset: offset,
			Items:  items,
		}, nil
	})
}
