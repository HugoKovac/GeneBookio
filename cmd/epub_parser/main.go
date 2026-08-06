package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"hkorpo/book/internal/book"
	"hkorpo/book/internal/platform/bucket"
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

	q, ch, err := queue.InitProducer(&cfg.ConfigQueue, primitive.Prepare)
	if err != nil {
		errorpkg.ExitTrace(err)
	}

	bucketServie := book.NewService(
		book.WithQueueRepo(book.NewQueueRepoImpl(q, ch)),
		book.WithBucketRepo(book.NewBucketRepoImpl(cClient)),
		book.WithEpubParser(book.NewEpubParserImpl()),
	)

	err = queue.InitConsumer(&cfg.ConfigQueue, primitive.Split, func(d amqp091.Delivery) error {
		fileName := string(d.Body)

		chunks, err := bucketServie.GetBookAsChunks(ctx, fileName)
		if err != nil {
			return err
		}

		if err := bucketServie.UploadBookChunks(ctx, fileName, chunks); err != nil {
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
