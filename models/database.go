package models

import (
	"os"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB
var redisClient *redis.Client
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

	redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		Password: "",
		DB: 0,
		Protocol: 2,
	})

}