package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"hkorpo/book/internal/book"
	"hkorpo/book/internal/parsing"
	"hkorpo/book/internal/platform/bucket"
	"hkorpo/book/internal/platform/database"
	"hkorpo/book/internal/platform/queue"
	"hkorpo/book/internal/primitive"
	"hkorpo/book/pkg/env"
	"hkorpo/book/pkg/errorpkg"

	"github.com/kelseyhightower/envconfig"
	"github.com/rabbitmq/amqp091-go"
)

type Config struct {
	queue.ConfigQueue
	bucket.ConfigBucket
	database.ConfigDB
}

func main() {
	var (
		cfg Config
		ctx = context.Background()
	)
	env.LoadEnv()

	if err := envconfig.Process("", &cfg); err != nil {
		errorpkg.ExitTrace(err)
	}

	cClient, err := bucket.Init(&cfg.ConfigBucket)
	if err != nil {
		errorpkg.ExitTrace(err)
	}

	dbClient, err := database.Init(&cfg.ConfigDB)
	if err != nil {
		errorpkg.ExitTrace(err)
	}

	q, ch, err := queue.InitProducer(&cfg.ConfigQueue, primitive.Prepare)
	if err != nil {
		errorpkg.ExitTrace(err)
	}

	repo := book.NewRepositoryImpl(dbClient)
	parsingService := parsing.NewService(
		repo,
		book.NewBucketRepoImpl(cClient),
		book.NewQueueRepoImpl(q, ch),
		parsing.NewEpubParserImpl(),
	)

	err = queue.InitConsumer(&cfg.ConfigQueue, primitive.Split, func(d amqp091.Delivery) error {
		bookID := string(d.Body)

		chunks, err := parsingService.GetBookAsChunks(ctx, bookID)
		if err != nil {
			return book.RecordFailure(ctx, repo, bookID, primitive.Split, err)
		}

		if err := parsingService.UploadBookChunks(ctx, bookID, chunks); err != nil {
			return book.RecordFailure(ctx, repo, bookID, primitive.Split, err)
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
