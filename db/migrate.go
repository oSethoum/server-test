package db

import "app/models"

func Migrate() {
	Client.AutoMigrate(
		&models.User{},
	)
}
