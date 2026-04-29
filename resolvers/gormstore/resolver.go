package gormstore

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/pid"
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
func NewResolver(db *gorm.DB, maxPIDAttempts int) backend.ResolverBackend {
	return &Store{db: db, maxPIDAttempts: maxPIDAttempts}
}

var _ backend.ResolverBackend = (*Store)(nil)

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

func (s *Store) ListNamespaces(ctx context.Context, params backend.ListNamespacesParams) (*backend.PaginatedNamespacesResponse, error) {
	return transaction(s.db.WithContext(ctx), func(tx *gorm.DB) (*backend.PaginatedNamespacesResponse, error) {
		countQ := tx.Model(&Namespace{})
		if params.Tag != nil {
			countQ = countQ.Where("tag = ?", *params.Tag)
		}
		var total int64
		if err := countQ.Count(&total).Error; err != nil {
			return nil, err
		}

		limit := params.Limit
		offset := params.Offset

		if int64(offset) >= total {
			return &backend.PaginatedNamespacesResponse{
				Total:  int(total),
				Offset: offset,
				Items:  []backend.NamespaceResponse{},
			}, nil
		}

		q := tx.Model(&Namespace{})
		if params.Tag != nil {
			q = q.Where("tag = ?", *params.Tag)
		}
		q = q.Order("namespace_uid")
		if limit >= 0 {
			q = q.Limit(limit)
		}
		q = q.Offset(offset)

		var rows []Namespace
		if err := q.Find(&rows).Error; err != nil {
			return nil, err
		}
		items := make([]backend.NamespaceResponse, len(rows))
		for i := range rows {
			items[i] = rows[i].ToApi()
		}
		return &backend.PaginatedNamespacesResponse{
			Total:  int(total),
			Offset: offset,
			Items:  items,
		}, nil
	})
}

func (s *Store) CreateNamespace(ctx context.Context, namespace string, req backend.NamespaceCreateRequest, now func() time.Time) (*backend.NamespaceResponse, error) {
	return transaction(s.db.WithContext(ctx), func(tx *gorm.DB) (*backend.NamespaceResponse, error) {
		ts := now().UTC()
		ns := Namespace{
			NamespaceUID: namespace,
			Tag:          req.Tag,
			PIDPattern:   req.PIDFormat.Pattern,
			PIDChars:     req.PIDFormat.Characters,
			DateCreated:  ts,
		}
		if err := tx.Create(&ns).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return nil, backend.ErrNamespaceIDAllocationFailed
			}
			return nil, err
		}
		resp := ns.ToApi()
		return &resp, nil
	})
}

func (s *Store) ListResources(ctx context.Context, params backend.ListResourcesParams) (*backend.PaginatedResourcesResponse, error) {
	return transaction(s.db.WithContext(ctx), func(tx *gorm.DB) (*backend.PaginatedResourcesResponse, error) {
		var ns Namespace
		if err := tx.Where("namespace_uid = ?", params.Namespace).First(&ns).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, backend.ErrNamespaceNotFound
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
			return &backend.PaginatedResourcesResponse{
				Total:  int(total),
				Offset: offset,
				Items:  []backend.ResourceResponse{},
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
		items := make([]backend.ResourceResponse, len(rows))
		for i := range rows {
			items[i] = rows[i].ToApi()
		}
		return &backend.PaginatedResourcesResponse{
			Total:  int(total),
			Offset: offset,
			Items:  items,
		}, nil
	})
}

func (s *Store) CreateResource(ctx context.Context, id string, req backend.ResourceCreateRequest, rand io.Reader, now func() time.Time) (*backend.ResourceResponse, error) {
	return transaction(s.db.WithContext(ctx), func(tx *gorm.DB) (*backend.ResourceResponse, error) {
		var ns Namespace
		if err := tx.Where("namespace_uid = ?", id).First(&ns).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, backend.ErrNamespaceNotFound
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
		return nil, backend.ErrPIDAllocationFailed
	})
}

func (s *Store) BatchCreateResources(ctx context.Context, id string, reqs []backend.ResourceCreateRequest, rand io.Reader, now func() time.Time) ([]backend.ResourceResponse, error) {
	if len(reqs) == 0 {
		return nil, nil
	}

	return transaction(s.db.WithContext(ctx), func(tx *gorm.DB) ([]backend.ResourceResponse, error) {
		var ns Namespace
		if err := tx.Where("namespace_uid = ?", id).First(&ns).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, backend.ErrNamespaceNotFound
			}
			return nil, err
		}
		out := make([]backend.ResourceResponse, 0, len(reqs))
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
				return nil, backend.ErrPIDAllocationFailed
			}
		}
		return out, nil
	})
}

func (s *Store) GetResource(ctx context.Context, id, pid string) (*backend.ResourceResponse, error) {
	return transaction(s.db.WithContext(ctx), func(tx *gorm.DB) (*backend.ResourceResponse, error) {
		var ns Namespace
		if err := tx.Where("namespace_uid = ?", id).First(&ns).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, backend.ErrNamespaceNotFound
			}
			return nil, err
		}
		var row Resource
		if err := tx.Where("namespace_id = ? AND pid = ?", ns.ID, pid).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, backend.ErrResourceNotFound
			}
			return nil, err
		}
		r := row.ToApi()
		return &r, nil
	})
}

func (s *Store) UpdateResource(ctx context.Context, id, pid string, req backend.ResourceUpdateRequest, now func() time.Time) (*backend.ResourceResponse, error) {
	return transaction(s.db.WithContext(ctx), func(tx *gorm.DB) (*backend.ResourceResponse, error) {
		var ns Namespace
		if err := tx.Where("namespace_uid = ?", id).First(&ns).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, backend.ErrNamespaceNotFound
			}
			return nil, err
		}
		var row Resource
		if err := tx.Where("namespace_id = ? AND pid = ?", ns.ID, pid).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, backend.ErrResourceNotFound
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
