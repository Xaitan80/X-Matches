package db

import (
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func Open(path string) *gorm.DB {
	d, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	return d
}
