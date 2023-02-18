package main

import (
	"fmt"
	"log"
	"os"

	"github.com/dacore-x/url-shortener/routes"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
)

// Setup routes for fiber app
func setupRoutes(app *fiber.App) {

	api := app.Group("/api")
	v1 := api.Group("/v1")
	v1.Get("/:url", routes.ResolveURL)
	v1.Post("/", routes.ShortenURL)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println(err)
	}

	// creating an instance of app
	app := fiber.New()

	// setting logging for app
	app.Use(logger.New())

	setupRoutes(app)

	// starting the server
	log.Fatal(app.Listen(os.Getenv("APP_PORT")))
}
