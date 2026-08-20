package main

import (
	"crypto/rsa"
	"os"

	"hkorpo/book/internal/book"
	"hkorpo/book/internal/catalog"
	"hkorpo/book/internal/library"
	"hkorpo/book/internal/platform/bucket"
	"hkorpo/book/internal/platform/queue"
	"hkorpo/book/internal/primitive"
	"hkorpo/book/internal/upload"
	"hkorpo/book/internal/user"
	"hkorpo/book/pkg/env"
	"hkorpo/book/pkg/errorpkg"

	"hkorpo/book/internal/platform/database"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	queue.ConfigQueue
	bucket.ConfigBucket
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
	var (
		cfg Config
		app = fiber.New()
		err error
	)
	env.LoadEnv()

	if err := envconfig.Process("", &cfg); err != nil {
		errorpkg.ExitTrace(err)
	}

	cfg.ConfigJWT.PrivateKey, cfg.ConfigJWT.PublicKey, err = readKeys(cfg.ConfigJWT.PRIVATE_KEY_PATH, cfg.ConfigJWT.PUBLIC_KEY_PATH)
	if err != nil {
		errorpkg.ExitTrace(err)
	}
	cfg.ConfigJWT.RefreshPrivateKey, cfg.ConfigJWT.RefreshPublicKey, err = readKeys(cfg.ConfigJWT.PRIVATE_REFRESH_KEY_PATH, cfg.ConfigJWT.PUBLIC_REFRESH_KEY_PATH)
	if err != nil {
		errorpkg.ExitTrace(err)
	}

	q, ch, err := queue.InitProducer(&cfg.ConfigQueue, primitive.Split)
	if err != nil {
		errorpkg.ExitTrace(err)
	}

	dbClient, err := database.Init(&cfg.ConfigDB)
	if err != nil {
		errorpkg.ExitTrace(err)
	}

	bucketClient, err := bucket.Init(&cfg.ConfigBucket)
	if err != nil {
		errorpkg.ExitTrace(err)
	}

	repo := book.NewRepositoryImpl(dbClient)
	bucketRepo := book.NewBucketRepoImpl(bucketClient)

	libraryService := library.NewService(library.NewOpenLibraryClient())
	catalogService := catalog.NewService(repo, bucketRepo)
	uploadService := upload.NewService(repo, bucketRepo, book.NewQueueRepoImpl(q, ch))
	userService := user.NewService(user.NewRepositoryImpl(dbClient), &cfg.ConfigJWT)

	upload.NewHandler(app, libraryService, catalogService, uploadService, userService)

	errorpkg.ExitTrace(app.Listen(":3001"))
}
