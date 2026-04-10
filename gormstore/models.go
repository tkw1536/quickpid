package gormstore

import "time"

// Namespace maps to the namespaces table.
type Namespace struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"size:255;uniqueIndex;not null"`
	DateCreated time.Time
	NextPID     int64 `gorm:"column:next_pid;default:0;not null"`
}

// Resource maps to the resources table.
type Resource struct {
	ID           uint   `gorm:"primaryKey"`
	NamespaceID uint `gorm:"not null;index:idx_ns_tag,priority:1;uniqueIndex:ux_ns_pid,priority:1"`
	PID         string `gorm:"column:pid;size:64;not null;uniqueIndex:ux_ns_pid,priority:2"`
	URL          string `gorm:"size:2048"`
	IdInTarget   string `gorm:"size:512"`
	DateCreated  time.Time
	DateUpdated  time.Time
	TargetSystem string `gorm:"size:255"`
	Tag          string `gorm:"size:255;index:idx_ns_tag,priority:2"`
	Deleted bool `gorm:"not null;default:false"`
}

func (Namespace) TableName() string {
	return "namespaces"
}

func (Resource) TableName() string {
	return "resources"
}
