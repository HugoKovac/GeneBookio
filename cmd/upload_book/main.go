package main

import (
	"errors"
	"hkorpo/book/internal/book"
	"hkorpo/book/internal/platform/queue"
	"hkorpo/book/internal/primitive"
	"log"
	"os"

	"github.com/gofiber/fiber/v3"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

func main() {
	var (
		cfg queue.ConfigQueue
		app = fiber.New()
	)

	if err := godotenv.Load("cmd/upload_book/.env"); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("load environment: %v", err)
	}

	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatal(err)
	}

	q, ch, err := queue.InitProducer(&cfg, primitive.Uploads)
	if err != nil {
		log.Fatal(err)
	}

	book.NewUploadHandlers(app, book.NewService(
		book.WithQueueRepo(book.NewQueueRepoImpl(q, ch)),
	))

	log.Fatal(app.Listen(":3001"))
}
