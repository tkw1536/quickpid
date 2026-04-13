package gormstore

import "gorm.io/gorm"

// Migrate creates or updates schema for Namespace and Resource, including indexes declared on models.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&Namespace{}, &Resource{})
}
