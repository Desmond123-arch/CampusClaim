package middleware

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Desmond123-arch/CampusClaim/internal/auth"
	"github.com/Desmond123-arch/CampusClaim/models"
	redisrate "github.com/go-redis/redis_rate/v10"
	"github.com/gofiber/fiber/v2"
)


var rateLimiter *redisrate.Limiter

func SetupRedisRateLimiter() {
	_, err := models.RedisClient.Ping(context.Background()).Result()

	if err != nil {
		log.Fatal("Error connecting to Redis:", err)
	}
	rateLimiter = redisrate.NewLimiter(models.RedisClient)
}

func AuthenticateMiddleware(c *fiber.Ctx) error {
	tokenString := c.Cookies("access-token")
	if tokenString != "" {
		c.Redirect("/login", fiber.StatusSeeOther)
		return fmt.Errorf("token is Required")
	}
	_, err := auth.VerifyToken(tokenString)
	if err != nil {
		c.Redirect("/login", fiber.StatusSeeOther)
		return err
	}
	c.Next()
	return nil
}

func VerifyRateLimiter(c *fiber.Ctx) error {
	ctx := context.Background()
	SetupRedisRateLimiter()
	res, err := rateLimiter.Allow(ctx, c.IP(), redisrate.Limit{
		Rate:   1,
		Burst:  1,
		Period: time.Second * 30,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Rate limiter error")
	}
	if res.Allowed <= 0 {
		// Handle rate limit exceeded error
		return c.SendStatus(fiber.StatusTooManyRequests)
	}
	return c.Next()
}
