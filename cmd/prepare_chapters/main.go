package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"hkorpo/book/internal/book"
	"hkorpo/book/internal/platform/bucket"
	"hkorpo/book/internal/platform/queue"
	"hkorpo/book/internal/primitive"
	"hkorpo/book/pkg/errorpkg"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/openai/openai-go/v3"
	"github.com/rabbitmq/amqp091-go"
)

type Config struct {
	queue.ConfigQueue
	bucket.ConfigBucket
}

func main() {
	var (
		cfg Config
		ctx = context.Background()
	)

	if err := godotenv.Load("cmd/prepare_chapters/.env"); err != nil {
		errorpkg.ExitTrace(fmt.Errorf("load environment: %v", err))
	}

	if err := envconfig.Process("", &cfg); err != nil {
		errorpkg.ExitTrace(err)
	}

	cClient, err := bucket.Init(&cfg.ConfigBucket)
	if err != nil {
		errorpkg.ExitTrace(err)
	}

	q, ch, err := queue.InitProducer(&cfg.ConfigQueue, primitive.Generate)
	if err != nil {
		errorpkg.ExitTrace(err)
	}

	bucketServie := book.NewService(
		book.WithQueueRepo(book.NewQueueRepoImpl(q, ch)),
		book.WithBucketRepo(book.NewBucketRepoImpl(cClient)),
		book.WithAiAPI(book.NewOpenAiClient(openai.NewClient())),
	)

	err = queue.InitConsumer(&cfg.ConfigQueue, primitive.Prepare, func(d amqp091.Delivery) error {
		fileName := string(d.Body)

		err = bucketServie.MapOnChunks(ctx, fileName, bucketServie.GenerateChapterPreparation)
		if err != nil {
			return err
		}
		return nil

	})
	if err != nil {
		errorpkg.ExitTrace(err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
}
