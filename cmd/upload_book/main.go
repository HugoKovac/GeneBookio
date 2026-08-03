package main

import (
	"hkorpo/book/internal/book"
	"hkorpo/book/internal/platform/bucket"
	"hkorpo/book/internal/platform/queue"
	"hkorpo/book/internal/primitive"
	"log"

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
		log.Fatalf("load environment: %v", err)
	}

	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatal(err)
	}

	q, ch, err := queue.InitProducer(&cfg.ConfigQueue, primitive.Uploads)
	if err != nil {
		log.Fatal(err)
	}

	bucketClient, err := bucket.Init(&cfg.ConfigBucket)
	if err != nil {
		log.Fatal(err)
	}

	book.NewUploadHandlers(app, book.NewService(
		book.WithQueueRepo(book.NewQueueRepoImpl(q, ch)),
		book.WithBucketRepo(book.NewBucketRepoImpl(bucketClient)),
	))

	log.Fatal(app.Listen(":3001"))
}
