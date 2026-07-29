package main

import (
	"log"

	"hkorpo/book/internal/book"
	"hkorpo/book/internal/platform/config"
	"hkorpo/book/internal/platform/database"
	"hkorpo/book/internal/platform/httpserver"
	"hkorpo/book/internal/user"

	_ "github.com/go-sql-driver/mysql"
)

func main() {

	config, err := config.Init()
	if err != nil {
		log.Fatal(err.Error())
	}

	dbClient, err := database.Init(&config.ConfigDB)
	if err != nil {
		log.Fatal(err.Error())
	}

	app := httpserver.Init()

	userRepo := user.NewRepositoryImpl(dbClient)
	userService := user.NewService(userRepo, &config.ConfigJWT)
	user.NewHandler(app.Group("/users"), userService)

	book.NewHandler(app.Group("/books"), book.NewService(book.NewOpenLibraryClient()))

	if err := app.Listen(":3000"); err != nil {
		log.Fatalf("fiber listen failed: %v", err)
	}
}
