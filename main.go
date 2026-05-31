package main

import (
	"log"

	"bookbuilder/internal/db"
	"bookbuilder/internal/routes"
	_ "bookbuilder/migrations"

	"github.com/pocketbase/pocketbase"
)

func main() {
	app := pocketbase.New()

	db.RegisterHooks(app)
	routes.Register(app)

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
