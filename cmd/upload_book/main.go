package main

import (
	"fmt"
	"hkorpo/book/internal/book"
	"hkorpo/book/internal/platform/bucket"
	"hkorpo/book/internal/platform/queue"
	"hkorpo/book/internal/primitive"
	"hkorpo/book/pkg/errorpkg"

	"github.com/gofiber/fiber/v3"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	queue.ConfigQueue
	bucket.ConfigBucket
}

func main() {
	var (
		cfg Config
		app = fiber.New()
	)

	if err := godotenv.Load("cmd/upload_book/.env"); err != nil {
		errorpkg.ExitTrace(fmt.Errorf("load environment: %v", err))
	}

	if err := envconfig.Process("", &cfg); err != nil {
		errorpkg.ExitTrace(err)
	}

	q, ch, err := queue.InitProducer(&cfg.ConfigQueue, primitive.Split)
	if err != nil {
		errorpkg.ExitTrace(err)
	}

	bucketClient, err := bucket.Init(&cfg.ConfigBucket)
	if err != nil {
		errorpkg.ExitTrace(err)
	}

	book.NewUploadHandlers(app, book.NewService(
		book.WithQueueRepo(book.NewQueueRepoImpl(q, ch)),
		book.WithBucketRepo(book.NewBucketRepoImpl(bucketClient)),
	))

	errorpkg.ExitTrace(app.Listen(":3001"))
}
