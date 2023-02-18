package routes

import (
	"github.com/dacore-x/url-shortener/database"
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
)

// ResolveURL redirecting user for original url of shorten url
func ResolveURL(c *fiber.Ctx) error {
	url := c.Params("url")

	// connecting to redis {"URL": "Short URL"} db
	shortenURLClient := database.CreateClient(0)
	defer shortenURLClient.Close()

	// getting original url by shorten url
	value, err := shortenURLClient.Get(database.Ctx, url).Result()

	if err == redis.Nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "url not found in database",
		})
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "cannot connect to db",
		})
	}
	
	return c.Redirect(value, 301)
}

