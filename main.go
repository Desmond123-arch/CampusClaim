package main

import (
	// "context"
	"log"
	"os"

	// "time"

	"github.com/Desmond123-arch/CampusClaim/internal/middleware"
	"github.com/Desmond123-arch/CampusClaim/models"
	"github.com/Desmond123-arch/CampusClaim/pkg"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/lpernett/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)



func main() {
	err := godotenv.Load()

	if err != nil {
		log.Fatal("Error loading .env file")
	}

	//POSTGRES SETUP
	postgres_url := os.Getenv("POSTGRES_URL")
	mongodb_url  := os.Getenv("MONGODB_URL")
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:postgres_url,
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
	if err != nil {
        panic("failed to connect database")
    }
	models.Setup(db)

	//MONGODB SETUP
	mongoDB, err := models.MongoSetup(mongodb_url)
	if err != nil {
		log.Fatalf("Error connecting to MongoDB: %v", err)
	}
	// message := models.Messages{
	// 	Sender:   1,
	// 	Receiver: 2,
	// 	Content:  "Hello",
	// 	SentAt:   time.Now(),
	// }
	// _, err = mongoDB.MessagesCollection.InsertOne(context.Background(), message)

	defer mongoDB.Close()


	app := fiber.New(fiber.Config{
		ErrorHandler: func (c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusBadRequest).JSON(pkg.GlobalErrorHandlerResp{
				Success: false,
				Message: err.Error(),
			})
		},
	})

	app.Use(logger.New())
	app.Use(middleware.AuthenticateMiddleware)
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})


	app.Listen(":3000")
}