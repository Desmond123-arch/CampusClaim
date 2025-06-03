package main

import (
	// "context"
	"log"
	"os"

	// "time"

	"github.com/Desmond123-arch/CampusClaim/internal/auth"
	// "github.com/Desmond123-arch/CampusClaim/internal/middleware"
	"github.com/Desmond123-arch/CampusClaim/models"
	"github.com/Desmond123-arch/CampusClaim/pkg"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/lpernett/godotenv"
	"gorm.io/gorm"
)

var DB *gorm.DB

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Fatal("Error loading .env file")
	}

	//POSTGRES SETUP
	mongodb_url  := os.Getenv("MONGODB_URL")

	models.Init()
	//MONGODB SETUP
	mongoDB, err := models.MongoSetup(mongodb_url)
	if err != nil {
		log.Fatalf("Error connecting to MongoDB: %v", err)
	}

	defer mongoDB.Close()


	app := fiber.New(fiber.Config{
		ErrorHandler: func (c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusBadRequest).JSON(pkg.GlobalErrorHandlerResp{
				Success: false,
				Message: err.Error(),
			})
		},
	})

	// app := fiber.New()
	// app.Use(middleware.AuthenticateMiddleware)
	app.Use(logger.New())

	authRoutes := app.Group("/auth")

	// AUTH ROUTES
	authRoutes.Post("/register", auth.RegisterUser)
	authRoutes.Post("/login", auth.LoginUser)
	authRoutes.Post("/verify-account", auth.VerifyAccount)
	app.Listen(":3000")
}