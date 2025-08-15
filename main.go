package main

import (
	// "context"
	"context"
	"log"
	"os"

	// "time"

	v1 "github.com/Desmond123-arch/CampusClaim/api/v1"
	"github.com/Desmond123-arch/CampusClaim/internal/auth"
	"github.com/Desmond123-arch/CampusClaim/internal/chat"
	"github.com/Desmond123-arch/CampusClaim/internal/firebase"
	"github.com/Desmond123-arch/CampusClaim/internal/middleware"

	// "github.com/Desmond123-arch/CampusClaim/internal/middleware"
	"github.com/Desmond123-arch/CampusClaim/models"
	"github.com/Desmond123-arch/CampusClaim/pkg"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/websocket/v2"
	"github.com/lpernett/godotenv"
	"gorm.io/gorm"
)

var DB *gorm.DB

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

}

func main() {

	models.Init()
	if err := firebase.Init(); err != nil {
		log.Fatal("Failed to initialize Firebase:", err)
	}
	defer models.MDB.Disconnect(context.Background())
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusBadRequest).JSON(pkg.GlobalErrorHandlerResp{
				Success: false,
				Message: err.Error(),
			})
		},
		BodyLimit: 50 * 1024 * 1024,
	})



	// app := fiber.New()
	// app.Use(middleware.AuthenticateMiddleware)
	app.Use(logger.New())

	app.Use(cors.New(cors.Config{
		AllowOrigins: os.Getenv("ALLOWED_ORIGIN"),
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS,PATCH",
		AllowHeaders: "Content-Type,Authorization,enctype",
		AllowCredentials: true,
	}))



	authRoutes := app.Group("/auth")
	profileRoutes := app.Group("/profile")
	itemsRoutes := app.Group("/items")
	claimRoutes := app.Group("/claims")

	// AUTH ROUTES
	authRoutes.Post("/register", auth.RegisterUser)
	authRoutes.Post("/login", auth.LoginUser)
	authRoutes.Post("/verify-account", middleware.AuthenticateMiddleware, middleware.VerifyRateLimiter, auth.VerifyAccount)
	authRoutes.Get("/refresh-token", auth.GetNewRefreshToken)

	authRoutes.Patch("/change-password", middleware.AuthenticateMiddleware, auth.ChangePassword)
	authRoutes.Post("/reset-password-request", auth.RequestPasswordreset)
	authRoutes.Post("/reset-password", auth.ResetPassword)
	authRoutes.Post("/reset-password-resend", middleware.AuthenticateMiddleware, auth.GetNewVerficationCode)

	//PROFILE ROUTES
	profileRoutes.Get("", middleware.AuthenticateMiddleware, v1.GetProfile)
	profileRoutes.Patch("", middleware.AuthenticateMiddleware, v1.UpdateProfile)
	profileRoutes.Delete("", middleware.AuthenticateMiddleware, v1.DeleteProfile)
	profileRoutes.Put("/token", middleware.AuthenticateMiddleware, v1.UpdateUserToken)

	//ITEMS_ROUTES
	itemsRoutes.Get("/my-items", middleware.AuthenticateMiddleware, v1.GetMyItems)
	itemsRoutes.Post("/search/image", middleware.AuthenticateMiddleware, v1.SearchByImage)
	itemsRoutes.Post("/search/text", middleware.AuthenticateMiddleware, v1.SearchByDescription)
	itemsRoutes.Get("", v1.GetItems)
	itemsRoutes.Get("/:id", v1.GetItem)
	itemsRoutes.Post("", middleware.AuthenticateMiddleware, v1.AddItem)
	itemsRoutes.Delete("/:id", middleware.AuthenticateMiddleware, v1.DeleteItem)
	itemsRoutes.Put("/:id", middleware.AuthenticateMiddleware, v1.UpdateItem)

	//CLAIM_ROUTES
	claimRoutes.Get("/:id", middleware.AuthenticateMiddleware, v1.GetItemCliams)
	claimRoutes.Post("/:id", middleware.AuthenticateMiddleware, v1.SubmitClaim)
	claimRoutes.Delete("/:id", middleware.AuthenticateMiddleware, v1.DeleteClaim)

	//CHAT AND WEBSOCKETS
	app.Get("/messages/:userId", middleware.AuthenticateMiddleware, v1.GetMessages)
	app.Get("/messages/:userId", middleware.AuthenticateMiddleware, v1.GetConversations)
	app.Get(
		"/ws",
		middleware.AuthenticateMiddleware,
		chat.WebSocketUpgradeMiddleware(),
		websocket.New(chat.HandleWebSocket),
	)
	log.Println("Starting server on :3000")
    if err := app.Listen(":3000"); err != nil {
        log.Fatal("Failed to start server:", err)
    }


}
