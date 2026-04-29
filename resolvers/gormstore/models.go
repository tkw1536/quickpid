package gormstore

import (
	"time"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/api/pid"
)

// Namespace maps to the namespaces table.
type Namespace struct {
	ID           uint             `gorm:"primaryKey"`
	NamespaceUID string           `gorm:"column:namespace_uid;type:text;not null;uniqueIndex"`
	Tag          string           `gorm:"type:text;not null;default:'';index"`
	PIDPattern   pid.Pattern      `gorm:"type:text;not null"`
	PIDChars     pid.CharacterSet `gorm:"type:text;not null"`
	DateCreated  time.Time
}

func (Namespace) TableName() string {
	return "namespaces"
}

func (n Namespace) ToApi() api.NamespaceResponse {
	return api.NamespaceResponse{
		ID:  n.NamespaceUID,
		Tag: n.Tag,
		PIDFormat: pid.Format{
			Pattern:    n.PIDPattern,
			Characters: n.PIDChars,
		},
		DateCreated: n.DateCreated.UTC().Format(time.RFC3339),
	}
}

// Resource maps to the resources table.
type Resource struct {
	ID          uint    `gorm:"primaryKey"`
	NamespaceID uint    `gorm:"not null;index:idx_ns_tag,priority:1;uniqueIndex:ux_ns_pid,priority:1"`
	PID         string  `gorm:"column:pid;type:text;not null;uniqueIndex:ux_ns_pid,priority:2"`
	URL         string  `gorm:"type:text"`
	Metadata    *string `gorm:"type:text"`
	DateCreated time.Time
	DateUpdated time.Time
	Tag         string `gorm:"type:text;index:idx_ns_tag,priority:2"`
	Deleted     bool   `gorm:"not null;default:false"`
}

func (Resource) TableName() string {
	return "resources"
}

func (r Resource) ToApi() api.ResourceResponse {
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
