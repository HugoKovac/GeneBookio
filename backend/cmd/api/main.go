// @title           Book Narration API
// @version         1.0
// @description     Queue-driven pipeline API that converts EPUBs into narrated audio.
// @host            localhost:3000
// @BasePath        /

// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Type "Bearer" followed by a space and the JWT access token.

package main

import (
	"crypto/rsa"
	"log"
	"os"

	"hkorpo/book/internal/book"
	"hkorpo/book/internal/catalog"
	"hkorpo/book/internal/library"
	"hkorpo/book/internal/platform/bucket"
	"hkorpo/book/internal/platform/database"
	"hkorpo/book/internal/platform/httpserver"
	"hkorpo/book/internal/user"
	"hkorpo/book/pkg/env"
	"hkorpo/book/pkg/errorpkg"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	database.ConfigDB
	user.ConfigJWT
	bucket.ConfigBucket
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
	var (
		cfg Config
		err error
	)
	env.LoadEnv()

	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatal(err)
	}

	cfg.ConfigJWT.PrivateKey, cfg.ConfigJWT.PublicKey, err = readKeys(cfg.ConfigJWT.PRIVATE_KEY_PATH, cfg.ConfigJWT.PUBLIC_KEY_PATH)
	cfg.ConfigJWT.RefreshPrivateKey, cfg.ConfigJWT.RefreshPublicKey, err = readKeys(cfg.ConfigJWT.PRIVATE_REFRESH_KEY_PATH, cfg.ConfigJWT.PUBLIC_REFRESH_KEY_PATH)

	dbClient, err := database.Init(&cfg.ConfigDB)
	if err != nil {
		log.Fatal(err)
	}

	cClient, err := bucket.Init(&cfg.ConfigBucket)
	if err != nil {
		errorpkg.ExitTrace(err)
	}

	app := httpserver.Init()

	userService := user.NewService(
		user.NewRepositoryImpl(dbClient),
		&cfg.ConfigJWT,
	)

	user.NewHandler(app.Group("/users"), userService)

	libraryService := library.NewService(library.NewOpenLibraryClient())
	catalogService := catalog.NewService(book.NewRepositoryImpl(dbClient), book.NewBucketRepoImpl(cClient))

	newBookHandlers(app.Group("/books"), libraryService, catalogService, userService)

	if err := app.Listen(":3000"); err != nil {
		log.Fatalf("fiber listen failed: %v", err)
	}
}
