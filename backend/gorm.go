package backend

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tkw1536/quickpid/pid"
	"github.com/tkw1536/quickpid/spec"
	"gorm.io/gorm"
)

// MigrateGorm automatically migrates the gorm schema used by [].
func MigrateGorm(db *gorm.DB) error {
	return db.AutoMigrate(&namespaceRow{}, &resourceRow{})
}

// DefaultGormBatchSize is the default batch size to be used by [NewGormBackend].
const DefaultGormBatchSize = 100

// NewGormBackend returns [Backend] backed by gorm.
//
// batchSize is the batch size to be used during create operations.
// If <= 0, [DefaultGormBatchSize] is used.
func NewGormBackend(db *gorm.DB, batchSize int) Backend {
	if batchSize <= 0 {
		batchSize = DefaultGormBatchSize
	}
	return &gormBackend{db: db, batchSize: batchSize}
}

// gormBackend implements api.Resolver using GORM. It is safe for concurrent use when backed by a
// database that serializes transactions appropriately (e.g. SQLite) or supports row locking.
type gormBackend struct {
	db *gorm.DB

	batchSize int // batch size to be used during create operations
}

func withTx[V any](db *gorm.DB, fn func(*gorm.DB) (V, error)) (V, error) {
	var out V
	if err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		out, err = fn(tx)
		return err
	}); err != nil {
		var zero V
		return zero, err
	}
	return out, nil
}

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}

	// some badly behaved drivers return strings ...
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint failed") ||
		strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "duplicated") ||
		strings.Contains(msg, "unique constraint")
}

func (s *gormBackend) ListNamespaces(ctx context.Context, params spec.ListNamespacesParams) (*spec.PaginatedNamespacesResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*spec.PaginatedNamespacesResponse, error) {
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
		if int64(offset) >= total {
			return &spec.PaginatedNamespacesResponse{
				Total:  int(total),
				Offset: offset,
				Items:  []spec.NamespaceResponse{},
			}, nil
		}

		var rows []namespaceRow
		if err := q.Order("id ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
			return nil, err
		}
		items := make([]spec.NamespaceResponse, len(rows))
		for i := range rows {
			items[i] = rows[i].toSpec()
		}
		return &spec.PaginatedNamespacesResponse{
			Total:  int(total),
			Offset: offset,
			Items:  items,
		}, nil
	})
}

func (s *gormBackend) CreateNamespace(ctx context.Context, namespace string, req spec.NamespaceCreateRequest, now func() time.Time) (*spec.NamespaceResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*spec.NamespaceResponse, error) {
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
				return nil, ErrDuplicateNamespaceID
			}
			return nil, err
		}
		resp := ns.toSpec()
		return &resp, nil
	})
}

func (s *gormBackend) GetNamespace(ctx context.Context, namespace string) (*spec.NamespaceResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*spec.NamespaceResponse, error) {
		var ns namespaceRow
		if err := tx.First(&ns, "id = ?", namespace).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrNamespaceNotFound
			}
			return nil, err
		}
		out := ns.toSpec()
		return &out, nil
	})
}

func (s *gormBackend) ListResources(ctx context.Context, params spec.ListResourcesParams) (*spec.PaginatedResourcesResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*spec.PaginatedResourcesResponse, error) {
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
		if int64(offset) >= total {
			return &spec.PaginatedResourcesResponse{
				Total:  int(total),
				Offset: offset,
				Items:  []spec.ResourceResponse{},
			}, nil
		}

		var rows []resourceRow
		if err := q.Order("pid ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
			return nil, err
		}
		items := make([]spec.ResourceResponse, len(rows))
		for i := range rows {
			items[i] = rows[i].toSpec()
		}
		return &spec.PaginatedResourcesResponse{
			Total:  int(total),
			Offset: offset,
			Items:  items,
		}, nil
	})
}

func (s *gormBackend) CreateResource(ctx context.Context, namespace, pid string, req spec.ResourceCreateRequest, now func() time.Time) (*spec.ResourceResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*spec.ResourceResponse, error) {
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
				return nil, ErrPIDAllocationFailed
			}
			return nil, err
		}
		r := row.toSpec()
		return &r, nil
	})
}

func (s *gormBackend) BatchCreateResources(ctx context.Context, namespace string, pids []string, reqs []spec.ResourceCreateRequest, now func() time.Time) ([]spec.ResourceResponse, error) {
	if len(reqs) == 0 {
		return nil, nil
	}
	if len(pids) != len(reqs) {
		return nil, ErrPIDAllocationFailed
	}

	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) ([]spec.ResourceResponse, error) {
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
				return nil, ErrPIDAllocationFailed
			}
			return nil, err
		}

		out := make([]spec.ResourceResponse, len(rows))
		for i := range rows {
			out[i] = rows[i].toSpec()
		}
		return out, nil
	})
}

func (s *gormBackend) GetResource(ctx context.Context, namespace, pid string) (*spec.ResourceResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*spec.ResourceResponse, error) {
		if err := ensureNamespaceExists(tx, namespace); err != nil {
			return nil, err
		}

		var row resourceRow
		if err := tx.First(&row, "namespace_id = ? AND pid = ?", namespace, pid).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrResourceNotFound
			}
			return nil, err
		}
		r := row.toSpec()
		return &r, nil
	})
}

func (s *gormBackend) UpdateResource(ctx context.Context, id, pid string, req spec.ResourceUpdateRequest, now func() time.Time) (*spec.ResourceResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*spec.ResourceResponse, error) {
		if err := ensureNamespaceExists(tx, id); err != nil {
			return nil, err
		}

		var row resourceRow
		if err := tx.First(&row, "namespace_id = ? AND pid = ?", id, pid).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrResourceNotFound
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

func ensureNamespaceExists(tx *gorm.DB, id string) error {
	var n int64
	if err := tx.Model(&namespaceRow{}).Where("id = ?", id).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return ErrNamespaceNotFound
	}
	return nil
}

// namespaceRow maps to the namespaces table.
type namespaceRow struct {
	ID string `gorm:"column:id;type:text;primaryKey"`

	DateCreated time.Time `gorm:"column:date_created;not null"`

	PIDPattern pid.Pattern      `gorm:"column:pid_pattern;type:text;not null"`
	PIDChars   pid.CharacterSet `gorm:"column:pid_chars;type:text;not null"`

	Tag string `gorm:"column:tag;type:text;not null;index"`
}

func (namespaceRow) TableName() string { return "namespaces" }

func (n namespaceRow) toSpec() spec.NamespaceResponse {
	return spec.NamespaceResponse{
		ID:  n.ID,
		Tag: n.Tag,
		PIDFormat: pid.Format{
			Pattern:    n.PIDPattern,
			Characters: n.PIDChars,
		},
		DateCreated: n.DateCreated.UTC().Format(time.RFC3339),
	}
}

// resourceRow maps to the resources table.
type resourceRow struct {
	NamespaceID string `gorm:"column:namespace_id;type:text;not null;primaryKey;index:idx_resources_namespace_pid,priority:1;index:idx_resources_ns_tag,priority:1"`
	PID         string `gorm:"column:pid;type:text;not null;primaryKey;index:idx_resources_namespace_pid,priority:2"`

	URL      string  `gorm:"column:url;type:text;not null"`
	Metadata *string `gorm:"column:metadata;type:text"`

	DateCreated time.Time `gorm:"column:date_created;not null"`
	DateUpdated time.Time `gorm:"column:date_updated;not null"`

	Tag     string `gorm:"column:tag;type:text;not null;index:idx_resources_ns_tag,priority:2"`
	Deleted bool   `gorm:"column:deleted;not null;default:false;index"`
}

func (resourceRow) TableName() string { return "resources" }

func (r resourceRow) toSpec() spec.ResourceResponse {
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

func (s *gormBackend) String() string {
	return fmt.Sprintf("gormBackend(batchSize=%d)", s.batchSize)
}
