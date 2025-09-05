package db

import (
	"log"

	"gorm.io/gorm"
)

func AutoMigrate(d *gorm.DB, models ...any) {
	if err := d.AutoMigrate(models...); err != nil {
		log.Fatalf("migrate: %v", err)
	}
}
