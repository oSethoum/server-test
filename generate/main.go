//go:build ignore
// +build ignore

package main

import (
	"app/models"

	"github.com/oSethoum/gorming"
	"github.com/oSethoum/gorming/types"
)

func main() {
	engine := gorming.New(types.Config{
		DBKind:      types.SQLite,
		Server:      types.Fiber,
		FilesAction: types.DoNotGenerate,
		Files: []types.File{
			types.DartApi,
			types.DartTypes,
			types.Images,
		},
		Paths: types.Paths{
			TypescriptClient: "client/api",
		},
	})
	engine([]any{
		models.User{},
	})
}
