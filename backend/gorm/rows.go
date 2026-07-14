//spellchecker:words gorm
package gorm

//spellchecker:words errors time github bicpid gorm
import (
	"errors"
	"time"

	"github.com/tkw1536/bicpid/api"
	"github.com/tkw1536/bicpid/pid"
	"gorm.io/gorm"
)

type namespaceRow struct {
	ID string `gorm:"column:id;type:text;primaryKey"`

	DateCreated time.Time `gorm:"column:date_created;not null"`

	PIDPattern pid.Pattern      `gorm:"column:pid_pattern;type:text;not null"`
	PIDChars   pid.CharacterSet `gorm:"column:pid_chars;type:text;not null"`

	Tag string `gorm:"column:tag;type:text;not null;index"`
}

func (namespaceRow) TableName() string { return "namespaces" }

func (n namespaceRow) toSpec() api.NamespaceResponse {
	return api.NamespaceResponse{
		ID:  n.ID,
		Tag: n.Tag,
		PIDFormat: pid.Format{
			Pattern:    n.PIDPattern,
			Characters: n.PIDChars,
		},
		DateCreated: n.DateCreated.UTC().Format(time.RFC3339),
	}
}

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

func (r resourceRow) toSpec() api.ResourceResponse {
	return api.ResourceResponse{
		PID:         r.PID,
		URL:         r.URL,
		Metadata:    r.Metadata,
		DateCreated: r.DateCreated.UTC().Format(time.RFC3339),
		DateUpdated: r.DateUpdated.UTC().Format(time.RFC3339),
		Tag:         r.Tag,
		Deleted:     r.Deleted,
	}
}

type userRow struct {
	Username  string `gorm:"column:username;type:text;primaryKey"`
	Superuser bool   `gorm:"column:superuser;not null;default:false"`
}

func (userRow) TableName() string { return "auth_users" }

func (u userRow) toSpec() api.UserInfo {
	return api.UserInfo{
		Username:  u.Username,
		Superuser: u.Superuser,
	}
}

type apiKeyRow struct {
	Username string `gorm:"column:username;type:text;not null;primaryKey;index:idx_auth_api_keys_user_id,priority:1"`
	ID       string `gorm:"column:id;type:text;not null;primaryKey;index:idx_auth_api_keys_user_id,priority:2"`

	Comment   string    `gorm:"column:comment;type:text;not null"`
	CreatedAt time.Time `gorm:"column:created_at;not null"`
	ExpiresAt *string   `gorm:"column:expires_at;type:text"`
	Revoked   bool      `gorm:"column:revoked;not null;default:false"`

	Prefix string `gorm:"column:prefix;type:text;not null;index"`
	Digest []byte `gorm:"column:digest;type:blob;not null"`
}

func (apiKeyRow) TableName() string { return "auth_api_keys" }

func (k apiKeyRow) toSpec() api.APIKeyInfo {
	return api.APIKeyInfo{
		ID:        k.ID,
		Comment:   k.Comment,
		CreatedAt: k.CreatedAt.UTC().Format(time.RFC3339),
		ExpiresAt: cloneStringPtr(k.ExpiresAt),
	}
}

type namespacePermissionRow struct {
	Namespace string `gorm:"column:namespace;type:text;not null;primaryKey"`
	Username  string `gorm:"column:username;type:text;not null;primaryKey"`
	Level     string `gorm:"column:level;type:text;not null"`
}

func (namespacePermissionRow) TableName() string { return "authz_namespace_permissions" }

func (p namespacePermissionRow) toSpec() api.NamespacePermission {
	return api.NamespacePermission{
		Username: p.Username,
		Level:    api.PermissionLevel(p.Level),
	}
}

func ensureNamespaceExists(tx *gorm.DB, id string) error {
	var n int64
	if err := tx.Model(&namespaceRow{}).Where("id = ?", id).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return errNamespaceNotFound
	}
	return nil
}

func ensureUserExists(tx *gorm.DB, username string) error {
	var n int64
	if err := tx.Model(&userRow{}).Where("username = ?", username).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return errUserNotFound
	}
	return nil
}

func findUser(tx *gorm.DB, username string) (userRow, error) {
	var row userRow
	if err := tx.First(&row, "username = ?", username).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return userRow{}, errUserNotFound
		}
		return userRow{}, err
	}
	return row, nil
}

func findKey(tx *gorm.DB, username, keyID string) (apiKeyRow, error) {
	var row apiKeyRow
	if err := tx.First(&row, "username = ? AND id = ? AND revoked = ?", username, keyID, false).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apiKeyRow{}, errKeyNotFound
		}
		return apiKeyRow{}, err
	}
	return row, nil
}

func findKeyIncludingRevoked(tx *gorm.DB, username, keyID string) (apiKeyRow, error) {
	var row apiKeyRow
	if err := tx.First(&row, "username = ? AND id = ?", username, keyID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apiKeyRow{}, errKeyNotFound
		}
		return apiKeyRow{}, err
	}
	return row, nil
}

func cloneStringPtr(v *string) *string {
	if v == nil {
		return nil
	}
	out := *v
	return &out
}
