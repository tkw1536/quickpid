//spellchecker:words gorm
package gorm

//spellchecker:words context errors time github quickpid backend gorm
import (
	"context"
	"errors"
	"time"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/pid"
	"gorm.io/gorm"
)

type namespaceRow struct {
	ID string `gorm:"column:id;type:text;primaryKey"`

	// Named Date* (not CreatedAt/UpdatedAt) so GORM does not auto-stamp wall clock.
	DateCreated time.Time `gorm:"column:created_at;not null"`
	DateUpdated time.Time `gorm:"column:updated_at;not null"`

	PIDPattern string `gorm:"column:pid_pattern;type:text;not null"`
	PIDChars   string `gorm:"column:pid_chars;type:text;not null"`

	TagRows   []namespaceTagRow  `gorm:"foreignKey:NamespaceID;references:ID;constraint:OnDelete:CASCADE"`
	Roles     []namespaceRoleRow `gorm:"foreignKey:NamespaceID;references:ID;constraint:OnDelete:CASCADE"`
	Mounts    []mountRow         `gorm:"foreignKey:NamespaceID;references:ID;constraint:OnDelete:CASCADE"`
	Resources []resourceRow      `gorm:"foreignKey:NamespaceID;references:ID;constraint:OnDelete:CASCADE"`
}

func (namespaceRow) TableName() string { return "namespaces" }

type namespaceTagRow struct {
	NamespaceID string `gorm:"column:namespace_id;type:text;not null;primaryKey;index:idx_namespace_tags_ns_tag,priority:1"`
	Pos         int    `gorm:"column:pos;not null;primaryKey"`
	Tag         string `gorm:"column:tag;type:text;not null;index:idx_namespace_tags_ns_tag,priority:2"`
}

func (namespaceTagRow) TableName() string { return "namespace_tags" }

func (n namespaceRow) toSpec() api.NamespaceResponse {
	return api.NamespaceResponse{
		ID:   n.ID,
		Tags: namespaceTagsFromRows(n.TagRows),
		PIDFormat: pid.Format{
			Pattern:    pid.Pattern(n.PIDPattern),
			Characters: pid.CharacterSet(n.PIDChars),
		},
		DateCreated: n.DateCreated.UTC(),
		DateUpdated: n.DateUpdated.UTC(),
	}
}

func namespaceTagRowsFor(namespaceID string, tags []string) []namespaceTagRow {
	rows := make([]namespaceTagRow, len(tags))
	for i, tag := range tags {
		rows[i] = namespaceTagRow{
			NamespaceID: namespaceID,
			Pos:         i,
			Tag:         tag,
		}
	}
	return rows
}

func namespaceTagsFromRows(rows []namespaceTagRow) []string {
	if len(rows) == 0 {
		return nil
	}
	tags := make([]string, len(rows))
	for i := range rows {
		tags[i] = rows[i].Tag
	}
	return tags
}

func ensureNamespaceExists(tx *gorm.DB, id api.ValidNamespaceID) error {
	var n int64
	if err := tx.Model(&namespaceRow{}).Where("id = ?", id.String()).Count(&n).Limit(1).Error; err != nil {
		return err
	}
	if n == 0 {
		return errNamespaceNotFound
	}
	return nil
}

func (s *GormBackend) ListNamespaces(ctx context.Context, params api.ListNamespacesParams) (*api.PaginatedNamespacesResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.PaginatedNamespacesResponse, error) {
		q := tx.Model(&namespaceRow{})
		if params.Tag != nil {
			q = q.Where(`EXISTS (
				SELECT 1 FROM namespace_tags
				WHERE namespace_tags.namespace_id = namespaces.id
					AND namespace_tags.tag = ?
			)`, *params.Tag)
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
		if err := q.Preload("TagRows", orderByPos).Order("namespaces.id ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
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

func (s *GormBackend) CreateNamespace(ctx context.Context, namespace api.ValidNamespaceID, req api.NamespaceCreateRequest, owner *api.ValidUsername, now func() time.Time) (*api.NamespaceResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.NamespaceResponse, error) {
		if owner != nil {
			if err := ensureUserExists(tx, *owner); err != nil {
				return nil, err
			}
		}

		ts := now().UTC()
		ns := namespaceRow{
			ID:         namespace.String(),
			PIDPattern: string(req.PIDFormat.Pattern),
			PIDChars:   string(req.PIDFormat.Characters),
			DateCreated: ts,
			DateUpdated: ts,
			TagRows:     namespaceTagRowsFor(namespace.String(), req.Tags),
		}
		if err := tx.Create(&ns).Error; err != nil {
			if isUniqueConstraintError(err) {
				return nil, backend.ErrDuplicateNamespaceID
			}
			return nil, err
		}

		if owner != nil {
			perm := namespaceRoleRow{
				NamespaceID: namespace.String(),
				Username:    owner.String(),
				Role:        string(api.RoleManager),
			}
			if err := tx.Create(&perm).Error; err != nil {
				return nil, err
			}
		}

		resp := ns.toSpec()
		return &resp, nil
	})
}

func (s *GormBackend) GetNamespace(ctx context.Context, namespace api.ValidNamespaceID) (*api.NamespaceResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.NamespaceResponse, error) {
		var ns namespaceRow
		if err := tx.Preload("TagRows", orderByPos).First(&ns, "id = ?", namespace.String()).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, backend.ErrNamespaceNotFound
			}
			return nil, err
		}
		out := ns.toSpec()
		return &out, nil
	})
}

func (s *GormBackend) UpdateNamespace(ctx context.Context, namespace api.ValidNamespaceID, req api.ValidNamespaceUpdateRequest, now func() time.Time) (*api.NamespaceResponse, error) {
	return withTx(s.db.WithContext(ctx), func(tx *gorm.DB) (*api.NamespaceResponse, error) {
		var ns namespaceRow
		if err := tx.Preload("TagRows", orderByPos).First(&ns, "id = ?", namespace.String()).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, backend.ErrNamespaceNotFound
			}
			return nil, err
		}
		if req.Tags != nil {
			if err := tx.Where("namespace_id = ?", namespace.String()).Delete(&namespaceTagRow{}).Error; err != nil {
				return nil, err
			}
			ns.TagRows = namespaceTagRowsFor(namespace.String(), req.Tags)
			if len(ns.TagRows) > 0 {
				if err := tx.Create(&ns.TagRows).Error; err != nil {
					return nil, err
				}
			}
		}
		ns.DateUpdated = now().UTC()
		if err := tx.Omit("TagRows", "Roles", "Mounts", "Resources").Save(&ns).Error; err != nil {
			return nil, err
		}
		out := ns.toSpec()
		return &out, nil
	})
}
