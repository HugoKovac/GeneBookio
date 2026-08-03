package main

import (
	"crypto/rsa"
	"log"
	"os"

	"hkorpo/book/internal/book"
	"hkorpo/book/internal/platform/database"
	"hkorpo/book/internal/platform/httpserver"
	"hkorpo/book/internal/user"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	database.ConfigDB
	user.ConfigJWT
}

func readKeys(privatePath, publicPath string) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateBytes, err := os.ReadFile(privatePath)
	if err != nil {
		return nil, nil, err
	}
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateBytes)
	if err != nil {
		return nil, nil, err
	}

	publicBytes, err := os.ReadFile(publicPath)
	if err != nil {
		return nil, nil, err
	}
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicBytes)
	if err != nil {
		return nil, nil, err
	}

	return privateKey, publicKey, nil
}

func main() {
	var config Config
	err := godotenv.Load("cmd/api/.env")
	if err != nil {
		log.Fatalf("load environment: %v", err)
	}

	if err := envconfig.Process("", &config); err != nil {
		log.Fatal(err)
	}

	config.ConfigJWT.PrivateKey, config.ConfigJWT.PublicKey, err = readKeys(config.ConfigJWT.PRIVATE_KEY_PATH, config.ConfigJWT.PUBLIC_KEY_PATH)
	config.ConfigJWT.RefreshPrivateKey, config.ConfigJWT.RefreshPublicKey, err = readKeys(config.ConfigJWT.PRIVATE_REFRESH_KEY_PATH, config.ConfigJWT.PUBLIC_REFRESH_KEY_PATH)

	dbClient, err := database.Init(&config.ConfigDB)
	if err != nil {
		log.Fatal(err)
	}

	app := httpserver.Init()

	userRepo := user.NewRepositoryImpl(dbClient)
	userService := user.NewService(userRepo, &config.ConfigJWT)
	user.NewHandler(app.Group("/users"), userService)

	book.NewHandlers(app.Group("/books"), book.NewService(
		book.WithLibraryAPI(book.NewOpenLibraryClient()),
	))

	if err := app.Listen(":3000"); err != nil {
		log.Fatalf("fiber listen failed: %v", err)
	}
}
