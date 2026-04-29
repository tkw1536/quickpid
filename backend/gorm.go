package backend

import (
	"context"
	"errors"
	"time"

	"github.com/tkw1536/quickpid/pid"
	"github.com/tkw1536/quickpid/spec"
	"gorm.io/gorm"
)

// MigrateGorm automatically migrates the gorm schema used by [].
func MigrateGorm(db *gorm.DB) error {
	return db.AutoMigrate(&namespaceModel{}, &resourceModel{})
}

// NewGormBackend returns [Backend] backed by gorm.
func NewGormBackend(db *gorm.DB) Backend {
	return &gormBackend{db: db}
}

// gormBackend implements api.Resolver using GORM. It is safe for concurrent use when backed by a
// database that serializes transactions appropriately (e.g. SQLite) or supports row locking.
type gormBackend struct {
	db *gorm.DB
}

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

func (s *gormBackend) ListNamespaces(ctx context.Context, params spec.ListNamespacesParams) (*spec.PaginatedNamespacesResponse, error) {
	return transaction(s.db.WithContext(ctx), func(tx *gorm.DB) (*spec.PaginatedNamespacesResponse, error) {
		countQ := tx.Model(&namespaceModel{})
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
			return &spec.PaginatedNamespacesResponse{
				Total:  int(total),
				Offset: offset,
				Items:  []spec.NamespaceResponse{},
			}, nil
		}

		q := tx.Model(&namespaceModel{})
		if params.Tag != nil {
			q = q.Where("tag = ?", *params.Tag)
		}
		q = q.Order("namespace_uid")
		if limit >= 0 {
			q = q.Limit(limit)
		}
		q = q.Offset(offset)

		var rows []namespaceModel
		if err := q.Find(&rows).Error; err != nil {
			return nil, err
		}
		items := make([]spec.NamespaceResponse, len(rows))
		for i := range rows {
			items[i] = rows[i].ToSpec()
		}
		return &spec.PaginatedNamespacesResponse{
			Total:  int(total),
			Offset: offset,
			Items:  items,
		}, nil
	})
}

func (s *gormBackend) CreateNamespace(ctx context.Context, namespace string, req spec.NamespaceCreateRequest, now func() time.Time) (*spec.NamespaceResponse, error) {
	return transaction(s.db.WithContext(ctx), func(tx *gorm.DB) (*spec.NamespaceResponse, error) {
		ts := now().UTC()
		ns := namespaceModel{
			NamespaceUID: namespace,
			Tag:          req.Tag,
			PIDPattern:   req.PIDFormat.Pattern,
			PIDChars:     req.PIDFormat.Characters,
			DateCreated:  ts,
		}
		if err := tx.Create(&ns).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return nil, ErrDuplicateNamespaceID
			}
			return nil, err
		}
		resp := ns.ToSpec()
		return &resp, nil
	})
}

func (s *gormBackend) GetNamespace(ctx context.Context, namespace string) (*spec.NamespaceResponse, error) {
	return transaction(s.db.WithContext(ctx), func(tx *gorm.DB) (*spec.NamespaceResponse, error) {
		var ns namespaceModel
		if err := tx.Where("namespace_uid = ?", namespace).First(&ns).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrNamespaceNotFound
			}
			return nil, err
		}
		out := ns.ToSpec()
		return &out, nil
	})
}

func (s *gormBackend) ListResources(ctx context.Context, params spec.ListResourcesParams) (*spec.PaginatedResourcesResponse, error) {
	return transaction(s.db.WithContext(ctx), func(tx *gorm.DB) (*spec.PaginatedResourcesResponse, error) {
		var ns namespaceModel
		if err := tx.Where("namespace_uid = ?", params.Namespace).First(&ns).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrNamespaceNotFound
			}
			return nil, err
		}

		q := tx.Model(&resourceModel{}).Where("namespace_id = ?", ns.ID)
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
			return &spec.PaginatedResourcesResponse{
				Total:  int(total),
				Offset: offset,
				Items:  []spec.ResourceResponse{},
			}, nil
		}

		q = q.Order("pid")
		if limit >= 0 {
			q = q.Limit(limit)
		}
		q = q.Offset(offset)

		var rows []resourceModel
		if err := q.Find(&rows).Error; err != nil {
			return nil, err
		}
		items := make([]spec.ResourceResponse, len(rows))
		for i := range rows {
			items[i] = rows[i].ToSpec()
		}
		return &spec.PaginatedResourcesResponse{
			Total:  int(total),
			Offset: offset,
			Items:  items,
		}, nil
	})
}

func (s *gormBackend) CreateResource(ctx context.Context, id, pid string, req spec.ResourceCreateRequest, now func() time.Time) (*spec.ResourceResponse, error) {
	return transaction(s.db.WithContext(ctx), func(tx *gorm.DB) (*spec.ResourceResponse, error) {
		var ns namespaceModel
		if err := tx.Where("namespace_uid = ?", id).First(&ns).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrNamespaceNotFound
			}
			return nil, err
		}

		ts := now().UTC()
		row := resourceModel{
			NamespaceID: ns.ID,
			PID:         pid,
			URL:         req.URL,
			Metadata:    req.Metadata,
			DateCreated: ts,
			DateUpdated: ts,
			Tag:         req.Tag,
			Deleted:     false,
		}
		if err := tx.Create(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return nil, ErrPIDAllocationFailed
			}
			return nil, err
		}
		r := row.ToSpec()
		return &r, nil
	})
}

func (s *gormBackend) BatchCreateResources(ctx context.Context, id string, pids []string, reqs []spec.ResourceCreateRequest, now func() time.Time) ([]spec.ResourceResponse, error) {
	if len(reqs) == 0 {
		return nil, nil
	}
	if len(pids) != len(reqs) {
		return nil, ErrPIDAllocationFailed
	}

	return transaction(s.db.WithContext(ctx), func(tx *gorm.DB) ([]spec.ResourceResponse, error) {
		var ns namespaceModel
		if err := tx.Where("namespace_uid = ?", id).First(&ns).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrNamespaceNotFound
			}
			return nil, err
		}
		out := make([]spec.ResourceResponse, 0, len(reqs))
		ts := now().UTC()
		for i, req := range reqs {
			row := resourceModel{
				NamespaceID: ns.ID,
				PID:         pids[i],
				URL:         req.URL,
				Metadata:    req.Metadata,
				DateCreated: ts,
				DateUpdated: ts,
				Tag:         req.Tag,
				Deleted:     false,
			}
			if err := tx.Create(&row).Error; err != nil {
				if errors.Is(err, gorm.ErrDuplicatedKey) {
					return nil, ErrPIDAllocationFailed
				}
				return nil, err
			}
			out = append(out, row.ToSpec())
		}
		return out, nil
	})
}

func (s *gormBackend) GetResource(ctx context.Context, id, pid string) (*spec.ResourceResponse, error) {
	return transaction(s.db.WithContext(ctx), func(tx *gorm.DB) (*spec.ResourceResponse, error) {
		var ns namespaceModel
		if err := tx.Where("namespace_uid = ?", id).First(&ns).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrNamespaceNotFound
			}
			return nil, err
		}
		var row resourceModel
		if err := tx.Where("namespace_id = ? AND pid = ?", ns.ID, pid).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrResourceNotFound
			}
			return nil, err
		}
		r := row.ToSpec()
		return &r, nil
	})
}

func (s *gormBackend) UpdateResource(ctx context.Context, id, pid string, req spec.ResourceUpdateRequest, now func() time.Time) (*spec.ResourceResponse, error) {
	return transaction(s.db.WithContext(ctx), func(tx *gorm.DB) (*spec.ResourceResponse, error) {
		var ns namespaceModel
		if err := tx.Where("namespace_uid = ?", id).First(&ns).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrNamespaceNotFound
			}
			return nil, err
		}
		var row resourceModel
		if err := tx.Where("namespace_id = ? AND pid = ?", ns.ID, pid).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrResourceNotFound
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
		r := row.ToSpec()
		return &r, nil
	})
}

// namespaceModel maps to the namespaces table.
type namespaceModel struct {
	ID uint `gorm:"primaryKey"`

	NamespaceUID string           `gorm:"column:namespace_id;type:text;not null;uniqueIndex"`
	Tag          string           `gorm:"type:text;not null;default:'';index"`
	PIDPattern   pid.Pattern      `gorm:"type:text;not null"`
	PIDChars     pid.CharacterSet `gorm:"type:text;not null"`
	DateCreated  time.Time
}

func (namespaceModel) TableName() string {
	return "namespaces"
}

func (n namespaceModel) ToSpec() spec.NamespaceResponse {
	return spec.NamespaceResponse{
		ID:  n.NamespaceUID,
		Tag: n.Tag,
		PIDFormat: pid.Format{
			Pattern:    n.PIDPattern,
			Characters: n.PIDChars,
		},
		DateCreated: n.DateCreated.UTC().Format(time.RFC3339),
	}
}

// resourceModel maps to the resources table.
type resourceModel struct {
	ID uint `gorm:"primaryKey"`

	NamespaceID uint    `gorm:"not null;index:idx_ns_tag,priority:1;uniqueIndex:ux_ns_pid,priority:1"`
	PID         string  `gorm:"column:pid;type:text;not null;uniqueIndex:ux_ns_pid,priority:2"`
	URL         string  `gorm:"type:text"`
	Metadata    *string `gorm:"type:text"`
	DateCreated time.Time
	DateUpdated time.Time
	Tag         string `gorm:"type:text;index:idx_ns_tag,priority:2"`
	Deleted     bool   `gorm:"not null;default:false"`
}

func (resourceModel) TableName() string {
	return "resources"
}

func (r resourceModel) ToSpec() spec.ResourceResponse {
	return spec.ResourceResponse{
		PID:         r.PID,
		URL:         r.URL,
		Metadata:    r.Metadata,
		DateCreated: r.DateCreated.UTC().Format(time.RFC3339),
		DateUpdated: r.DateUpdated.UTC().Format(time.RFC3339),
		Tag:         r.Tag,
		Deleted:     r.Deleted,
	}
}
