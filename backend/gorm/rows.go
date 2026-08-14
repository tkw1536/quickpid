//spellchecker:words gorm
package gorm

//spellchecker:words errors sort time github quickpid gorm
import (
	"errors"
	"sort"
	"time"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/pid"
	"gorm.io/gorm"
)

type namespaceRow struct {
	ID string `gorm:"column:id;type:text;primaryKey"`

	DateCreated time.Time `gorm:"column:date_created;not null"`
	DateUpdated time.Time `gorm:"column:date_updated;not null"`

	PIDPattern pid.Pattern      `gorm:"column:pid_pattern;type:text;not null"`
	PIDChars   pid.CharacterSet `gorm:"column:pid_chars;type:text;not null"`

	TagRows []namespaceTagRow `gorm:"foreignKey:NamespaceID;references:ID;constraint:OnDelete:CASCADE"`
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
			Pattern:    n.PIDPattern,
			Characters: n.PIDChars,
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
	sorted := append([]namespaceTagRow(nil), rows...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Pos < sorted[j].Pos })
	tags := make([]string, len(sorted))
	for i := range sorted {
		tags[i] = sorted[i].Tag
	}
	return tags
}

func replaceNamespaceTags(tx *gorm.DB, namespaceID string, tags []string) error {
	if err := tx.Where("namespace_id = ?", namespaceID).Delete(&namespaceTagRow{}).Error; err != nil {
		return err
	}
	if len(tags) == 0 {
		return nil
	}
	return tx.Create(namespaceTagRowsFor(namespaceID, tags)).Error
}

func preloadNamespaceTags(db *gorm.DB) *gorm.DB {
	return db.Order("pos ASC")
}

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

type userRow struct {
	Username     string `gorm:"column:username;type:text;primaryKey"`
	Superuser    bool   `gorm:"column:superuser;not null;default:false"`
	PasswordHash []byte `gorm:"column:password_hash"`
}

func (userRow) TableName() string { return "auth_users" }

func (u userRow) toSpec() api.UserInfo {
	return api.UserInfo{
		Username:  u.Username,
		Superuser: u.Superuser,
		Password:  len(u.PasswordHash) > 0,
	}
}

type apiKeyRow struct {
	Username string `gorm:"column:username;type:text;not null;primaryKey;index:idx_auth_api_keys_user_id,priority:1"`
	ID       string `gorm:"column:id;type:text;not null;primaryKey;index:idx_auth_api_keys_user_id,priority:2"`

	Comment   string     `gorm:"column:comment;type:text;not null"`
	CreatedAt time.Time  `gorm:"column:created_at;not null"`
	ExpiresAt *time.Time `gorm:"column:expires_at"`

	Prefix string `gorm:"column:prefix;type:text;not null;index"`
	Digest []byte `gorm:"column:digest;not null"`
}

func (apiKeyRow) TableName() string { return "auth_api_keys" }

func (k apiKeyRow) toSpec() api.APIKeyInfo {
	expiresAt := k.ExpiresAt
	if expiresAt != nil {
		utc := expiresAt.UTC()
		expiresAt = &utc
	}
	return api.APIKeyInfo{
		ID:        k.ID,
		Comment:   k.Comment,
		CreatedAt: k.CreatedAt.UTC(),
		ExpiresAt: expiresAt,
	}
}

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

type mountRow struct {
	BaseURI     string `gorm:"column:base_uri;type:text;primaryKey"`
	NamespaceID string `gorm:"column:namespace_id;type:text;not null;index"`
}

func (mountRow) TableName() string { return "mounts" }

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

func ensureUserExists(tx *gorm.DB, user api.ValidUsername) error {
	var n int64
	if err := tx.Model(&userRow{}).Where("username = ?", user.String()).Count(&n).Limit(1).Error; err != nil {
		return err
	}
	if n == 0 {
		return errUserNotFound
	}
	return nil
}

func findUser(tx *gorm.DB, username api.ValidUsername) (userRow, error) {
	var row userRow
	if err := tx.First(&row, "username = ?", username.String()).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return userRow{}, errUserNotFound
		}
		return userRow{}, err
	}
	return row, nil
}
