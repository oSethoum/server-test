package routes

import "github.com/gofiber/fiber/v2"

func Init(r fiber.Router) {
	routes(r.Group("api"))
}
