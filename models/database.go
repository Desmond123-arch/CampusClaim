package models

import (
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init() {
	postgres_url := os.Getenv("POSTGRES_URL")
	var err error
	DB, err = gorm.Open(postgres.New(postgres.Config{
		DSN:postgres_url,
		PreferSimpleProtocol: true,
	}), &gorm.Config{ TranslateError: true})

	if err != nil {
        panic("failed to connect database")
    }
	Setup(DB)
}