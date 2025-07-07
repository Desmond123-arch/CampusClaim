package main

import (
	// "context"
	"context"
	"log"

	// "time"

	v1 "github.com/Desmond123-arch/CampusClaim/api/v1"
	"github.com/Desmond123-arch/CampusClaim/internal/auth"
	"github.com/Desmond123-arch/CampusClaim/internal/chat"
	"github.com/Desmond123-arch/CampusClaim/internal/middleware"

	// "github.com/Desmond123-arch/CampusClaim/internal/middleware"
	"github.com/Desmond123-arch/CampusClaim/models"
	"github.com/Desmond123-arch/CampusClaim/pkg"
	"github.com/gofiber/fiber/v2"
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
	defer models.MDB.Disconnect(context.Background())
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
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
	profileRoutes := app.Group("/profile")
	itemsRoutes := app.Group("/items")
	claimRoutes := app.Group("/claims")
	// AUTH ROUTES
	authRoutes.Post("/register", auth.RegisterUser)
	authRoutes.Post("/login", auth.LoginUser)
	authRoutes.Post("/verify-account", middleware.VerifyRateLimiter, auth.VerifyAccount)
	authRoutes.Get("/refresh-token", auth.GetNewRefreshToken)

	authRoutes.Put("/change-password", middleware.AuthenticateMiddleware, auth.ChangePassword)
	authRoutes.Post("/reset-password-request", auth.RequestPasswordreset)
	authRoutes.Post("/reset-password", auth.ResetPassword)

	//PROFILE ROUTES
	profileRoutes.Get("", middleware.AuthenticateMiddleware, v1.GetProfile)
	profileRoutes.Patch("", middleware.AuthenticateMiddleware, v1.UpdateProfile)
	profileRoutes.Delete("", middleware.AuthenticateMiddleware, v1.DeleteProfile)

	//ITEMS_ROUTES
	itemsRoutes.Get("", v1.GetItems)
	itemsRoutes.Get("/:id", v1.GetItem)
	itemsRoutes.Get("/my-items", middleware.AuthenticateMiddleware, v1.GetMyItems)
	itemsRoutes.Post("", middleware.AuthenticateMiddleware, v1.AddItem)
	itemsRoutes.Delete("/:id", middleware.AuthenticateMiddleware, v1.DeleteItem)
	itemsRoutes.Put("/:id", middleware.AuthenticateMiddleware, v1.UpdateItem)

	//CLAIM_ROUTES
	claimRoutes.Get("/:id", middleware.AuthenticateMiddleware, v1.GetItemCliams)
	claimRoutes.Post("/:id", middleware.AuthenticateMiddleware, v1.SubmitClaim)
	claimRoutes.Delete("/:id", middleware.AuthenticateMiddleware, v1.DeleteClaim)

	//CHAT AND WEBSOCKETS
	app.Get("/messages/:userId", middleware.AuthenticateMiddleware, v1.GetMessages)
	app.Get(
		"/ws",
		middleware.AuthenticateMiddleware,
		chat.WebSocketUpgradeMiddleware(),
		websocket.New(chat.HandleWebSocket),
	  )
	  
	app.Listen(":3000")

}
