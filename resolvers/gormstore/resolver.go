package gormstore

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/api/pid"
	"gorm.io/gorm"
)

// Store implements api.Resolver using GORM. It is safe for concurrent use when backed by a
// database that serializes transactions appropriately (e.g. SQLite) or supports row locking.
type Store struct {
	db             *gorm.DB
	maxPIDAttempts int
}

// NewResolver returns an api.Resolver backed by db. The caller must open db with any GORM dialector.
// PID allocation is supplied per CreateResource / BatchCreateResources via pidGen; on duplicate key,
// the store calls pidGen again at most maxPIDAttempts times per row.
func NewResolver(db *gorm.DB, maxPIDAttempts int) api.Resolver {
	return &Store{db: db, maxPIDAttempts: maxPIDAttempts}
}

var _ api.Resolver = (*Store)(nil)

func transaction[V any](db *gorm.DB, fn func(*gorm.DB) (V, error)) (V, error) {
	var out V
	err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		out, err = fn(tx)
		return err
	})
	if err != nil {
		var zero V
		return zero, err
	}
	return out, nil
}

func (s *Store) ListNamespaces(ctx context.Context, params api.ListNamespacesParams) (*api.PaginatedNamespacesResponse, error) {
	return transaction(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.PaginatedNamespacesResponse, error) {
		var total int64
		if err := tx.Model(&Namespace{}).Count(&total).Error; err != nil {
			return nil, err
		}

		limit := params.Limit
		offset := params.Offset

		if int64(offset) >= total {
			return &api.PaginatedNamespacesResponse{
				Total:  int(total),
				Offset: offset,
				Items:  []api.NamespaceResponse{},
			}, nil
		}

		q := tx.Order("name")
		if limit >= 0 {
			q = q.Limit(limit)
		}
		q = q.Offset(offset)

		var rows []Namespace
		if err := q.Find(&rows).Error; err != nil {
			return nil, err
		}
		items := make([]api.NamespaceResponse, len(rows))
		for i := range rows {
			items[i] = rows[i].ToApi()
		}
		return &api.PaginatedNamespacesResponse{
			Total:  int(total),
			Offset: offset,
			Items:  items,
		}, nil
	})
}

func (s *Store) CreateNamespace(ctx context.Context, req api.NamespaceCreateRequest, now func() time.Time) (*api.NamespaceResponse, error) {
	return transaction(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.NamespaceResponse, error) {
		var existing Namespace
		err := tx.Where("name = ?", req.Name).First(&existing).Error
		if err == nil {
			return nil, api.ErrNamespaceAlreadyExists
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		ts := now().UTC()
		ns := Namespace{
			Name:        req.Name,
			PIDPattern:  req.PIDFormat.Pattern,
			PIDChars:    req.PIDFormat.Characters,
			DateCreated: ts,
		}
		if err := tx.Create(&ns).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return nil, api.ErrNamespaceAlreadyExists
			}
			return nil, err
		}
		resp := ns.ToApi()
		return &resp, nil
	})
}

func (s *Store) ListResources(ctx context.Context, params api.ListResourcesParams) (*api.PaginatedResourcesResponse, error) {
	return transaction(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.PaginatedResourcesResponse, error) {
		var ns Namespace
		if err := tx.Where("name = ?", params.Namespace).First(&ns).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, api.ErrNamespaceNotFound
			}
			return nil, err
		}

		q := tx.Model(&Resource{}).Where("namespace_id = ?", ns.ID)
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

		if int64(offset) >= total {
			return &api.PaginatedResourcesResponse{
				Total:  int(total),
				Offset: offset,
				Items:  []api.ResourceResponse{},
			}, nil
		}

		q = q.Order("pid")
		if limit >= 0 {
			q = q.Limit(limit)
		}
		q = q.Offset(offset)

		var rows []Resource
		if err := q.Find(&rows).Error; err != nil {
			return nil, err
		}
		items := make([]api.ResourceResponse, len(rows))
		for i := range rows {
			items[i] = rows[i].ToApi()
		}
		return &api.PaginatedResourcesResponse{
			Total:  int(total),
			Offset: offset,
			Items:  items,
		}, nil
	})
}

func (s *Store) CreateResource(ctx context.Context, namespace string, req api.ResourceCreateRequest, rand io.Reader, now func() time.Time) (*api.ResourceResponse, error) {
	return transaction(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.ResourceResponse, error) {
		var ns Namespace
		if err := tx.Where("name = ?", namespace).First(&ns).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, api.ErrNamespaceNotFound
			}
			return nil, err
		}

		for attempt := 0; attempt < s.maxPIDAttempts; attempt++ {
			candidate, err := pid.Format{
				Pattern:    ns.PIDPattern,
				Characters: ns.PIDChars,
			}.Generate(rand)
			if err != nil {
				return nil, err
			}
			ts := now().UTC()
			row := Resource{
				NamespaceID: ns.ID,
				PID:         candidate,
				URL:         req.URL,
				Metadata:    req.Metadata,
				DateCreated: ts,
				DateUpdated: ts,
				Tag:         req.Tag,
				Deleted:     false,
			}
			if err := tx.Create(&row).Error; err != nil {
				if errors.Is(err, gorm.ErrDuplicatedKey) {
					continue
				}
				return nil, err
			}
			r := row.ToApi()
			return &r, nil
		}
		return nil, api.ErrPIDAllocationFailed
	})
}

func (s *Store) BatchCreateResources(ctx context.Context, namespace string, reqs []api.ResourceCreateRequest, rand io.Reader, now func() time.Time) ([]api.ResourceResponse, error) {
	if len(reqs) == 0 {
		return nil, nil
	}

	return transaction(s.db.WithContext(ctx), func(tx *gorm.DB) ([]api.ResourceResponse, error) {
		var ns Namespace
		if err := tx.Where("name = ?", namespace).First(&ns).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, api.ErrNamespaceNotFound
			}
			return nil, err
		}
		out := make([]api.ResourceResponse, 0, len(reqs))
		for _, req := range reqs {
			var inserted bool
			for attempt := 0; attempt < s.maxPIDAttempts; attempt++ {
				candidate, err := pid.Format{
					Pattern:    ns.PIDPattern,
					Characters: ns.PIDChars,
				}.Generate(rand)
				if err != nil {
					return nil, err
				}
				ts := now().UTC()
				row := Resource{
					NamespaceID: ns.ID,
					PID:         candidate,
					URL:         req.URL,
					Metadata:    req.Metadata,
					DateCreated: ts,
					DateUpdated: ts,
					Tag:         req.Tag,
					Deleted:     false,
				}
				if err := tx.Create(&row).Error; err != nil {
					if errors.Is(err, gorm.ErrDuplicatedKey) {
						continue
					}
					return nil, err
				}
				out = append(out, row.ToApi())
				inserted = true
				break
			}
			if !inserted {
				return nil, api.ErrPIDAllocationFailed
			}
		}
		return out, nil
	})
}

func (s *Store) GetResource(ctx context.Context, namespace, pid string) (*api.ResourceResponse, error) {
	return transaction(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.ResourceResponse, error) {
		var ns Namespace
		if err := tx.Where("name = ?", namespace).First(&ns).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, api.ErrNamespaceNotFound
			}
			return nil, err
		}
		var row Resource
		if err := tx.Where("namespace_id = ? AND pid = ?", ns.ID, pid).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, api.ErrResourceNotFound
			}
			return nil, err
		}
		r := row.ToApi()
		return &r, nil
	})
}

func (s *Store) UpdateResource(ctx context.Context, namespace, pid string, req api.ResourceUpdateRequest, now func() time.Time) (*api.ResourceResponse, error) {
	return transaction(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.ResourceResponse, error) {
		var ns Namespace
		if err := tx.Where("name = ?", namespace).First(&ns).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, api.ErrNamespaceNotFound
			}
			return nil, err
		}
		var row Resource
		if err := tx.Where("namespace_id = ? AND pid = ?", ns.ID, pid).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, api.ErrResourceNotFound
			}
			return nil, err
		}
		ts := now().UTC()
		row.URL = req.URL
		row.Metadata = req.Metadata
		row.Tag = req.Tag
		row.Deleted = req.Deleted
		row.DateUpdated = ts
		if err := tx.Save(&row).Error; err != nil {
			return nil, err
		}
		r := row.ToApi()
		return &r, nil
	})
}
