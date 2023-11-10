package db

import (
	"log"

	"github.com/oSethoum/sqlite"

	"gorm.io/gorm"
)

var Client *gorm.DB

func Init() {
	dialect := sqlite.Open("file:db.sqlite?_fk=1")
	client, err := gorm.Open(dialect, &gorm.Config{
		PrepareStmt: true})

	if err != nil {
		log.Fatalln(err)
	}
	Client = client
}

func Close() {
	conn, _ := Client.DB()
	conn.Close()
}
