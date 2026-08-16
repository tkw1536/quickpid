//spellchecker:words gorm
package gorm

//spellchecker:words context errors sort time github quickpid backend gorm
import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
	"gorm.io/gorm"
)

type resourceRow struct {
	NamespaceID string `gorm:"column:namespace_id;type:text;not null;primaryKey;index:idx_resources_namespace_pid,priority:1"`
	PID         string `gorm:"column:pid;type:text;not null;primaryKey;index:idx_resources_namespace_pid,priority:2"`

	URL      *string `gorm:"column:url;type:text"`
	Metadata *string `gorm:"column:metadata;type:text"`

	DateCreated time.Time `gorm:"column:date_created;not null"`
	DateUpdated time.Time `gorm:"column:date_updated;not null"`

	Deleted bool `gorm:"column:deleted;not null;default:false;index"`

	TagRows []resourceTagRow `gorm:"foreignKey:NamespaceID,PID;references:NamespaceID,PID;constraint:OnDelete:CASCADE"`
}

func (resourceRow) TableName() string { return "resources" }

type resourceTagRow struct {
	NamespaceID string `gorm:"column:namespace_id;type:text;not null;primaryKey;index:idx_resource_tags_ns_tag,priority:1"`
	PID         string `gorm:"column:pid;type:text;not null;primaryKey"`
	Pos         int    `gorm:"column:pos;not null;primaryKey"`
	Tag         string `gorm:"column:tag;type:text;not null;index:idx_resource_tags_ns_tag,priority:2"`
}

func (resourceTagRow) TableName() string { return "resource_tags" }

func (r resourceRow) toSpec() api.ResourceResponse {
	return api.ResourceResponse{
		PID:         r.PID,
		URL:         r.URL,
		Metadata:    r.Metadata,
		DateCreated: r.DateCreated.UTC().Format(time.RFC3339),
		DateUpdated: r.DateUpdated.UTC().Format(time.RFC3339),
		Tags:        tagsFromRows(r.TagRows),
		Deleted:     r.Deleted,
	}
}

func tagRowsFor(namespaceID, pid string, tags []string) []resourceTagRow {
	rows := make([]resourceTagRow, len(tags))
	for i, tag := range tags {
		rows[i] = resourceTagRow{
			NamespaceID: namespaceID,
			PID:         pid,
			Pos:         i,
			Tag:         tag,
		}
	}
	return rows
}

func tagsFromRows(rows []resourceTagRow) []string {
	if len(rows) == 0 {
		return nil
	}
	sorted := append([]resourceTagRow(nil), rows...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Pos < sorted[j].Pos })
	tags := make([]string, len(sorted))
	for i := range sorted {
		tags[i] = sorted[i].Tag
	}
	return tags
}

func replaceResourceTags(tx *gorm.DB, namespaceID, pid string, tags []string) error {
	if err := tx.Where("namespace_id = ? AND pid = ?", namespaceID, pid).Delete(&resourceTagRow{}).Error; err != nil {
		return err
	}
	if len(tags) == 0 {
		return nil
	}
	return tx.Create(tagRowsFor(namespaceID, pid, tags)).Error
}

func preloadResourceTags(db *gorm.DB) *gorm.DB {
	return db.Order("pos ASC")
}

func (s *Store) ListResources(ctx context.Context, namespace api.ValidNamespaceID, params api.ListResourcesParams) (*api.PaginatedResourcesResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.PaginatedResourcesResponse, error) {
		if err := ensureNamespaceExists(tx, namespace); err != nil {
			return nil, err
		}

		q := tx.Model(&resourceRow{}).Where("namespace_id = ?", namespace.String())
		if params.Tag != nil {
			q = q.Where(`EXISTS (
				SELECT 1 FROM resource_tags
				WHERE resource_tags.namespace_id = resources.namespace_id
					AND resource_tags.pid = resources.pid
					AND resource_tags.tag = ?
			)`, *params.Tag)
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
		if err := q.Preload("TagRows", preloadResourceTags).Order("pid ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
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

func (s *Store) CreateResource(ctx context.Context, namespace api.ValidNamespaceID, pid api.ValidPID, req api.ValidResourceCreateRequest, now func() time.Time) (*api.ResourceResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.ResourceResponse, error) {
		if err := ensureNamespaceExists(tx, namespace); err != nil {
			return nil, err
		}
		ts := now().UTC()
		row := resourceRow{
			NamespaceID: namespace.String(),
			PID:         pid.String(),
			URL:         req.URL.StringPtr(),
			Metadata:    req.Metadata,
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
		tagRows := tagRowsFor(namespace.String(), pid.String(), req.Tags)
		if len(tagRows) > 0 {
			if err := tx.Create(&tagRows).Error; err != nil {
				return nil, err
			}
		}
		row.TagRows = tagRows
		r := row.toSpec()
		return &r, nil
	})
}

func (s *Store) BatchCreateResources(ctx context.Context, namespace api.ValidNamespaceID, pids []api.ValidPID, reqs []api.ValidResourceCreateRequest, now func() time.Time) ([]api.ResourceResponse, error) {
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
		var allTags []resourceTagRow
		for i, req := range reqs {
			rows[i] = resourceRow{
				NamespaceID: namespace.String(),
				PID:         pids[i].String(),
				URL:         req.URL.StringPtr(),
				Metadata:    req.Metadata,
				Deleted:     false,
				DateCreated: ts,
				DateUpdated: ts,
			}
			allTags = append(allTags, tagRowsFor(namespace.String(), pids[i].String(), req.Tags)...)
		}

		if err := tx.CreateInBatches(&rows, s.batchSize).Error; err != nil {
			if isUniqueConstraintError(err) {
				return nil, backend.ErrPIDAllocationFailed
			}
			return nil, err
		}
		if len(allTags) > 0 {
			if err := tx.CreateInBatches(&allTags, s.batchSize).Error; err != nil {
				return nil, err
			}
		}

		out := make([]api.ResourceResponse, len(rows))
		for i := range rows {
			rows[i].TagRows = tagRowsFor(namespace.String(), pids[i].String(), reqs[i].Tags)
			out[i] = rows[i].toSpec()
		}
		return out, nil
	})
}

func (s *Store) GetResource(ctx context.Context, namespace api.ValidNamespaceID, pid api.ValidPID) (*api.ResourceResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.ResourceResponse, error) {
		if err := ensureNamespaceExists(tx, namespace); err != nil {
			return nil, err
		}

		var row resourceRow
		if err := tx.Preload("TagRows", preloadResourceTags).First(&row, "namespace_id = ? AND pid = ?", namespace.String(), pid.String()).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, backend.ErrResourceNotFound
			}
			return nil, err
		}
		r := row.toSpec()
		return &r, nil
	})
}

func (s *Store) UpdateResource(ctx context.Context, namespace api.ValidNamespaceID, pid api.ValidPID, req api.ValidResourceUpdateRequest, now func() time.Time) (*api.ResourceResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.ResourceResponse, error) {
		if err := ensureNamespaceExists(tx, namespace); err != nil {
			return nil, err
		}

		var row resourceRow
		if err := tx.Preload("TagRows", preloadResourceTags).First(&row, "namespace_id = ? AND pid = ?", namespace.String(), pid.String()).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, backend.ErrResourceNotFound
			}
			return nil, err
		}

		if req.URL != nil {
			row.URL = (*req.URL).StringPtr()
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
		if req.Tags != nil {
			if err := replaceResourceTags(tx, namespace.String(), pid.String(), req.Tags); err != nil {
				return nil, err
			}
			row.TagRows = tagRowsFor(namespace.String(), pid.String(), req.Tags)
		}
		r := row.toSpec()
		return &r, nil
	})
}
