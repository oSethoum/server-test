package handlers

import (
	"app/db"
	"app/models"
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func QueryResource[T models.Model](resource string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		result := new([]T)
		query := new(db.Query)
		q := c.Query("query", "{}")
		err := json.Unmarshal([]byte(q), query)
		if err != nil {
			return Error(c, err)
		}
		client, err := query.P(db.Client.WithContext(c.UserContext()), resource)
		if err != nil {
			return Error(c, err)
		}
		var count int64
		err = client.Find(result).Count(&count).Error
		if err != nil {
			return Error(c, err)
		}
		data := fiber.Map{"count": count, "result": result}
		go Broadcast(resource+":query", result)
		return Success(c, data)
	}
}

func CreateResource[T models.Model](resource string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		body := new([]T)
		err := c.BodyParser(body)
		if err != nil {
			return Error(c, err)
		}
		err = db.Client.WithContext(c.UserContext()).Transaction(func(tx *gorm.DB) error {
			for i, v := range *body {
				if err := tx.Create(&v).Error; err != nil {
					return ApiResponseError{MainError: err, Index: i}
				}
				(*body)[i] = v
			}
			return nil
		})
		if err != nil {
			return Error(c, err)
		}
		go Broadcast(resource+":create", body)
		return Success(c, body, fiber.StatusCreated)
	}
}

func UpdateResource[T models.Model](resource string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		body := new([]T)
		err := c.BodyParser(body)
		if err != nil {
			return Error(c, err)
		}
		db.Client.WithContext(c.UserContext()).Transaction(func(tx *gorm.DB) error {
			for i, v := range *body {
				err := tx.Updates(&v).Error
				if err != nil {
					return ApiResponseError{MainError: err, Index: i}
				}
				(*body)[i] = v
			}
			return nil
		})
		if err != nil {
			return Error(c, err)
		}
		go Broadcast(resource+"update", body)
		return Success(c, body)
	}
}

func DeleteResource[T models.Model](resource string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		body := new(db.Where)
		err := c.BodyParser(body)
		if err != nil {
			return Error(c, err)
		}
		predicate, vars, err := body.P()
		if err != nil {
			if err != nil {
				return Error(c, err)
			}
		}
		client := db.Client.WithContext(c.UserContext()).Model(new(T))
		data := new([]T)
		client.Where(predicate, vars...).Find(data)
		client.Delete(predicate, vars...)
		err = client.Delete(predicate, vars...).Error
		if err != nil {
			return Error(c, err)
		}
		go Broadcast(resource+":delete", data)
		return Success(c, data)
	}
}
