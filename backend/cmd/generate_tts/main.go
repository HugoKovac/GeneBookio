package main

import (
	"context"
	_ "embed"
	"os"
	"os/signal"
	"syscall"

	"hkorpo/book/internal/book"
	"hkorpo/book/internal/platform/bucket"
	"hkorpo/book/internal/platform/database"
	"hkorpo/book/internal/platform/queue"
	"hkorpo/book/internal/platform/ttsapi"
	"hkorpo/book/internal/primitive"
	"hkorpo/book/internal/tts"
	"hkorpo/book/pkg/env"
	"hkorpo/book/pkg/errorpkg"

	"github.com/kelseyhightower/envconfig"
	"github.com/rabbitmq/amqp091-go"
)

type Config struct {
	queue.ConfigQueue
	bucket.ConfigBucket
	database.ConfigDB
	ttsapi.ConfigTTSAPI
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

	repo := book.NewRepositoryImpl(dbClient)
	ttsService := tts.NewService(
		repo,
		book.NewBucketRepoImpl(cClient),
		tts.NewTTSApiClient(ttsapi.Init(&cfg.ConfigTTSAPI)),
	)

	err = queue.InitConsumer(&cfg.ConfigQueue, primitive.GenerateTTS, func(d amqp091.Delivery) error {
		bookID := string(d.Body)

		if err := ttsService.CreateAudioFromScript(ctx, bookID); err != nil {
			return book.RecordFailure(ctx, repo, bookID, primitive.GenerateTTS, err)
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
