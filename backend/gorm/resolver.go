package gorm

//spellchecker:words context errors fmt time github bicpid backend gorm
import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/tkw1536/bicpid/api"
	"github.com/tkw1536/bicpid/backend"
	"gorm.io/gorm"
)

var errShutdownGormStore = errors.New("stopped waiting for shutdown to complete")

func (s *Store) Shutdown(ctx context.Context) error {
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
		return fmt.Errorf("%w: %w", errShutdownGormStore, ctx.Err())
	case err := <-result:
		return err
	}
}

func (s *Store) ListNamespaces(ctx context.Context, params api.ListNamespacesParams) (*api.PaginatedNamespacesResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.PaginatedNamespacesResponse, error) {
		q := tx.Model(&namespaceRow{})
		if params.Tag != nil {
			q = q.Where("tag = ?", *params.Tag)
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
		if err := q.Order("id ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
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

func (s *Store) CreateNamespace(ctx context.Context, namespace string, req api.NamespaceCreateRequest, owner string, now func() time.Time) (*api.NamespaceResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.NamespaceResponse, error) {
		if err := ensureUserExists(tx, owner); err != nil {
			return nil, err
		}

		ts := now().UTC()
		ns := namespaceRow{
			ID:          namespace,
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

		perm := namespacePermissionRow{
			Namespace: namespace,
			Username:  owner,
			Level:     string(api.PermissionLevelManager),
		}
		if err := tx.Create(&perm).Error; err != nil {
			return nil, err
		}

		resp := ns.toSpec()
		return &resp, nil
	})
}

func (s *Store) GetNamespace(ctx context.Context, namespace string) (*api.NamespaceResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.NamespaceResponse, error) {
		var ns namespaceRow
		if err := tx.First(&ns, "id = ?", namespace).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, backend.ErrNamespaceNotFound
			}
			return nil, err
		}
		out := ns.toSpec()
		return &out, nil
	})
}

func (s *Store) ListResources(ctx context.Context, params api.ListResourcesParams) (*api.PaginatedResourcesResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.PaginatedResourcesResponse, error) {
		if err := ensureNamespaceExists(tx, params.Namespace); err != nil {
			return nil, err
		}

		q := tx.Model(&resourceRow{}).Where("namespace_id = ?", params.Namespace)
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

func (s *Store) CreateResource(ctx context.Context, namespace, pid string, req api.ResourceCreateRequest, now func() time.Time) (*api.ResourceResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.ResourceResponse, error) {
		if err := ensureNamespaceExists(tx, namespace); err != nil {
			return nil, err
		}
		ts := now().UTC()
		row := resourceRow{
			NamespaceID: namespace,
			PID:         pid,
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

func (s *Store) BatchCreateResources(ctx context.Context, namespace string, pids []string, reqs []api.ResourceCreateRequest, now func() time.Time) ([]api.ResourceResponse, error) {
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
				NamespaceID: namespace,
				PID:         pids[i],
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

func (s *Store) GetResource(ctx context.Context, namespace, pid string) (*api.ResourceResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.ResourceResponse, error) {
		if err := ensureNamespaceExists(tx, namespace); err != nil {
			return nil, err
		}

		var row resourceRow
		if err := tx.First(&row, "namespace_id = ? AND pid = ?", namespace, pid).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, backend.ErrResourceNotFound
			}
			return nil, err
		}
		r := row.toSpec()
		return &r, nil
	})
}

func (s *Store) UpdateResource(ctx context.Context, id, pid string, req api.ResourceUpdateRequest, now func() time.Time) (*api.ResourceResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.ResourceResponse, error) {
		if err := ensureNamespaceExists(tx, id); err != nil {
			return nil, err
		}

		var row resourceRow
		if err := tx.First(&row, "namespace_id = ? AND pid = ?", id, pid).Error; err != nil {
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
