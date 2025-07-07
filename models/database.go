package models

import (
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB
var RedisClient *redis.Client
var MDB *mongo.Client
func Init() {
	mongodb_url := os.Getenv("MONGODB_URL")

	var err error
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"),
		os.Getenv("POSTGRES_PORT"),
	)

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{ TranslateError: true})

	//setup for categories

	if err != nil {
        panic("failed to connect database")
    }
	Setup(DB)
	MDB, err = MongoSetup(mongodb_url)
	fmt.Println(mongodb_url)
	if err != nil {
		fmt.Println(err)
        panic("failed to connect to mongo database")
    }
	RedisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		Password: "",
		DB: 0,
		Protocol: 2,
	})

}

