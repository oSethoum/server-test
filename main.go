package main

import (
	"app/db"
	"app/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	db.Init()
	db.Migrate()
	app := fiber.New()
	app.Use(cors.New())
	app.Use(logger.New())
	routes.Init(app)
	app.Listen(":5000")
}
