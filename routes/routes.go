package routes

import (
	"app/handlers"
	"app/models"

	"github.com/gofiber/fiber/v2"
)

func routes(r fiber.Router) {
	users := r.Group("/users")
	users.Get("/", handlers.QueryResource[models.User]("users"))
	users.Post("/", handlers.CreateResource[models.User]("users"))
	users.Patch("/", handlers.UpdateResource[models.User]("users"))
	users.Delete("/", handlers.DeleteResource[models.User]("users"))

}
