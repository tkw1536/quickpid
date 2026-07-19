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

func (s *Store) ListNamespaces(ctx context.Context, user *api.ValidUsername, params api.ListNamespacesParams) (*api.PaginatedNamespacesResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.PaginatedNamespacesResponse, error) {
		if user != nil {
			if err := ensureUserExists(tx, user); err != nil {
				return nil, err
			}
		}

		q := tx.Model(&namespaceRow{})
		if user != nil {
			q = q.Joins("INNER JOIN authz_namespace_permissions ON authz_namespace_permissions.namespace = namespaces.id").
				Where("authz_namespace_permissions.username = ?", user.String())
		}
		if params.Tag != nil {
			q = q.Where("namespaces.tag = ?", *params.Tag)
		}

		var total int64
		if err := q.Count(&total).Error; err != nil {
			return nil, err
		}

		limit := params.Limit
		offset := params.Offset
		if limit == 0 || int64(offset) >= total {
			return &api.PaginatedNamespacesResponse{
				Total:  int(total),
				Offset: offset,
				Items:  []api.NamespaceResponse{},
			}, nil
		}

		var rows []namespaceRow
		if err := q.Order("namespaces.id ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
			return nil, err
		}
		items := make([]api.NamespaceResponse, len(rows))
		for i := range rows {
			items[i] = rows[i].toSpec()
		}
		return &api.PaginatedNamespacesResponse{
			Total:  int(total),
			Offset: offset,
			Items:  items,
		}, nil
	})
}

func (s *Store) CreateNamespace(ctx context.Context, namespace *api.ValidNamespaceID, req api.NamespaceCreateRequest, owner *api.ValidUsername, now func() time.Time) (*api.NamespaceResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.NamespaceResponse, error) {
		if owner != nil {
			if err := ensureUserExists(tx, owner); err != nil {
				return nil, err
			}
		}

		ts := now().UTC()
		ns := namespaceRow{
			ID:          namespace.String(),
			Tag:         req.Tag,
			PIDPattern:  req.PIDFormat.Pattern,
			PIDChars:    req.PIDFormat.Characters,
			DateCreated: ts,
		}
		if err := tx.Create(&ns).Error; err != nil {
			if isUniqueConstraintError(err) {
				return nil, backend.ErrDuplicateNamespaceID
			}
			return nil, err
		}

		if owner != nil {
			perm := namespacePermissionRow{
				Namespace: namespace.String(),
				Username:  owner.String(),
				Level:     string(api.PermissionLevelManager),
			}
			if err := tx.Create(&perm).Error; err != nil {
				return nil, err
			}
		}

		resp := ns.toSpec()
		return &resp, nil
	})
}

func (s *Store) GetNamespace(ctx context.Context, namespace *api.ValidNamespaceID) (*api.NamespaceResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.NamespaceResponse, error) {
		var ns namespaceRow
		if err := tx.First(&ns, "id = ?", namespace.String()).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, backend.ErrNamespaceNotFound
			}
			return nil, err
		}
		out := ns.toSpec()
		return &out, nil
	})
}

func (s *Store) ListResources(ctx context.Context, namespace *api.ValidNamespaceID, params api.ListResourcesParams) (*api.PaginatedResourcesResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.PaginatedResourcesResponse, error) {
		if err := ensureNamespaceExists(tx, namespace); err != nil {
			return nil, err
		}

		q := tx.Model(&resourceRow{}).Where("namespace_id = ?", namespace.String())
		if params.Tag != nil {
			q = q.Where("tag = ?", *params.Tag)
		}
		if params.Deleted != nil {
			q = q.Where("deleted = ?", *params.Deleted)
		}

		var total int64
		if err := q.Count(&total).Error; err != nil {
			return nil, err
		}

		limit := params.Limit
		offset := params.Offset
		if limit == 0 || int64(offset) >= total {
			return &api.PaginatedResourcesResponse{
				Total:  int(total),
				Offset: offset,
				Items:  []api.ResourceResponse{},
			}, nil
		}

		var rows []resourceRow
		if err := q.Order("pid ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
			return nil, err
		}
		items := make([]api.ResourceResponse, len(rows))
		for i := range rows {
			items[i] = rows[i].toSpec()
		}
		return &api.PaginatedResourcesResponse{
			Total:  int(total),
			Offset: offset,
			Items:  items,
		}, nil
	})
}

func (s *Store) CountAllResources(ctx context.Context) (int64, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (int64, error) {
		var total int64
		if err := tx.Model(&resourceRow{}).Count(&total).Error; err != nil {
			return 0, err
		}
		return total, nil
	})
}

func (s *Store) CreateResource(ctx context.Context, namespace *api.ValidNamespaceID, pid *api.ValidPID, req api.ResourceCreateRequest, now func() time.Time) (*api.ResourceResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.ResourceResponse, error) {
		if err := ensureNamespaceExists(tx, namespace); err != nil {
			return nil, err
		}
		ts := now().UTC()
		row := resourceRow{
			NamespaceID: namespace.String(),
			PID:         pid.String(),
			URL:         req.URL,
			Metadata:    req.Metadata,
			Tag:         req.Tag,
			Deleted:     false,
			DateCreated: ts,
			DateUpdated: ts,
		}
		if err := tx.Create(&row).Error; err != nil {
			if isUniqueConstraintError(err) {
				return nil, backend.ErrPIDAllocationFailed
			}
			return nil, err
		}
		r := row.toSpec()
		return &r, nil
	})
}

func (s *Store) BatchCreateResources(ctx context.Context, namespace *api.ValidNamespaceID, pids []*api.ValidPID, reqs []api.ResourceCreateRequest, now func() time.Time) ([]api.ResourceResponse, error) {
	if len(reqs) == 0 {
		return nil, nil
	}
	if len(pids) != len(reqs) {
		return nil, backend.ErrPIDAllocationFailed
	}

	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) ([]api.ResourceResponse, error) {
		if err := ensureNamespaceExists(tx, namespace); err != nil {
			return nil, err
		}
		ts := now().UTC()
		rows := make([]resourceRow, len(reqs))
		for i, req := range reqs {
			rows[i] = resourceRow{
				NamespaceID: namespace.String(),
				PID:         pids[i].String(),
				URL:         req.URL,
				Metadata:    req.Metadata,
				Tag:         req.Tag,
				Deleted:     false,
				DateCreated: ts,
				DateUpdated: ts,
			}
		}

		if err := tx.CreateInBatches(&rows, s.batchSize).Error; err != nil {
			if isUniqueConstraintError(err) {
				return nil, backend.ErrPIDAllocationFailed
			}
			return nil, err
		}

		out := make([]api.ResourceResponse, len(rows))
		for i := range rows {
			out[i] = rows[i].toSpec()
		}
		return out, nil
	})
}

func (s *Store) GetResource(ctx context.Context, namespace *api.ValidNamespaceID, pid *api.ValidPID) (*api.ResourceResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.ResourceResponse, error) {
		if err := ensureNamespaceExists(tx, namespace); err != nil {
			return nil, err
		}

		var row resourceRow
		if err := tx.First(&row, "namespace_id = ? AND pid = ?", namespace.String(), pid.String()).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, backend.ErrResourceNotFound
			}
			return nil, err
		}
		r := row.toSpec()
		return &r, nil
	})
}

func (s *Store) UpdateResource(ctx context.Context, namespace *api.ValidNamespaceID, pid *api.ValidPID, req api.ResourceUpdateRequest, now func() time.Time) (*api.ResourceResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.ResourceResponse, error) {
		if err := ensureNamespaceExists(tx, namespace); err != nil {
			return nil, err
		}

		var row resourceRow
		if err := tx.First(&row, "namespace_id = ? AND pid = ?", namespace.String(), pid.String()).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, backend.ErrResourceNotFound
			}
			return nil, err
		}

		if req.URL != nil {
			row.URL = *req.URL
		}
		if req.Tag != nil {
			row.Tag = *req.Tag
		}
		if req.Deleted != nil {
			row.Deleted = *req.Deleted
		}
		if req.Metadata != nil {
			row.Metadata = *req.Metadata
		}
		row.DateUpdated = now().UTC()
		if err := tx.Save(&row).Error; err != nil {
			return nil, err
		}
		r := row.toSpec()
		return &r, nil
	})
}
