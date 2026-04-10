package gormstore

import "time"

// Namespace maps to the namespaces table.
type Namespace struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"type:text;uniqueIndex;not null"`
	DateCreated time.Time
}

// Resource maps to the resources table.
type Resource struct {
	ID           uint   `gorm:"primaryKey"`
	NamespaceID  uint   `gorm:"not null;index:idx_ns_tag,priority:1;uniqueIndex:ux_ns_pid,priority:1"`
	PID          string `gorm:"column:pid;type:text;not null;uniqueIndex:ux_ns_pid,priority:2"`
	URL          string `gorm:"type:text"`
	IdInTarget   string `gorm:"type:text"`
	DateCreated  time.Time
	DateUpdated  time.Time
	TargetSystem string `gorm:"type:text"`
	Tag          string `gorm:"type:text;index:idx_ns_tag,priority:2"`
	Deleted      bool   `gorm:"not null;default:false"`
}

func (Namespace) TableName() string {
	return "namespaces"
}

func (Resource) TableName() string {
	return "resources"
}
