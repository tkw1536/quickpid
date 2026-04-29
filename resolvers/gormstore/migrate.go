package gormstore

import (
	"fmt"

	"gorm.io/gorm"
)

// Migrate creates or updates schema for Namespace and Resource, including indexes declared on models.
// It migrates legacy SQLite rows that used column `name` as the namespace path segment into `namespace_uid`.
func Migrate(db *gorm.DB) error {
	if err := migrateNamespacesLegacy(db); err != nil {
		return err
	}
	return db.AutoMigrate(&Namespace{}, &Resource{})
}

func migrateNamespacesLegacy(db *gorm.DB) error {
	if db.Dialector.Name() != "sqlite" {
		return nil
	}

	var tableExists int64
	if err := db.Raw(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='namespaces'`).Scan(&tableExists).Error; err != nil {
		return err
	}
	if tableExists == 0 {
		return nil
	}

	var hasName int64
	if err := db.Raw(`SELECT COUNT(*) FROM pragma_table_info('namespaces') WHERE name='name'`).Scan(&hasName).Error; err != nil {
		return err
	}
	if hasName == 0 {
		return nil
	}

	var hasUID int64
	if err := db.Raw(`SELECT COUNT(*) FROM pragma_table_info('namespaces') WHERE name='namespace_uid'`).Scan(&hasUID).Error; err != nil {
		return err
	}
	if hasUID == 0 {
		if err := db.Exec(`ALTER TABLE namespaces ADD COLUMN namespace_uid TEXT`).Error; err != nil {
			return fmt.Errorf("gormstore migrate: add namespace_uid: %w", err)
		}
	}
	if err := db.Exec(`UPDATE namespaces SET namespace_uid = name WHERE namespace_uid IS NULL OR namespace_uid = ''`).Error; err != nil {
		return fmt.Errorf("gormstore migrate: backfill namespace_uid: %w", err)
	}

	var hasTag int64
	if err := db.Raw(`SELECT COUNT(*) FROM pragma_table_info('namespaces') WHERE name='tag'`).Scan(&hasTag).Error; err != nil {
		return err
	}
	if hasTag == 0 {
		if err := db.Exec(`ALTER TABLE namespaces ADD COLUMN tag TEXT NOT NULL DEFAULT ''`).Error; err != nil {
			return fmt.Errorf("gormstore migrate: add tag: %w", err)
		}
	}

	if err := db.Exec(`ALTER TABLE namespaces DROP COLUMN name`).Error; err != nil {
		return fmt.Errorf("gormstore migrate: drop legacy name: %w", err)
	}
	return nil
}
