package main

import (
	"hkorpo/book/internal/book"
	"hkorpo/book/internal/catalog"
	"hkorpo/book/internal/library"
	"hkorpo/book/internal/platform/bucket"
	"hkorpo/book/internal/platform/queue"
	"hkorpo/book/internal/primitive"
	"hkorpo/book/internal/upload"
	"hkorpo/book/pkg/env"
	"hkorpo/book/pkg/errorpkg"

	"hkorpo/book/internal/platform/database"

	"github.com/gofiber/fiber/v3"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	queue.ConfigQueue
	bucket.ConfigBucket
	database.ConfigDB
}

func main() {
	var (
		cfg Config
		app = fiber.New()
	)
	env.LoadEnv()

	if err := envconfig.Process("", &cfg); err != nil {
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

	upload.NewHandler(app, libraryService, catalogService, uploadService)

	errorpkg.ExitTrace(app.Listen(":3001"))
}
