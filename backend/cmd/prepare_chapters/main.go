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
	"hkorpo/book/internal/preparation"
	"hkorpo/book/internal/pricing"
	"hkorpo/book/internal/primitive"
	"hkorpo/book/pkg/env"
	"hkorpo/book/pkg/errorpkg"

	"github.com/kelseyhightower/envconfig"
	"github.com/openai/openai-go/v3"
	"github.com/rabbitmq/amqp091-go"
)

type Config struct {
	queue.ConfigQueue
	bucket.ConfigBucket
	database.ConfigDB
	book.ConfigAi
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

	q, ch, err := queue.InitProducer(&cfg.ConfigQueue, primitive.GenerateScript)
	if err != nil {
		errorpkg.ExitTrace(err)
	}

	var aiAPI preparation.AiAPI
	if cfg.ConfigAi.TEST_MODE {
		aiAPI = book.NewSubstitutionAiClient()
	} else {
		aiAPI = book.NewOpenAiClient(openai.NewClient(), openai.ChatModelGPT5Mini)
	}

	repo := book.NewRepositoryImpl(dbClient)
	preparationService := preparation.NewService(
		repo,
		book.NewBucketRepoImpl(cClient),
		book.NewQueueRepoImpl(q, ch),
		aiAPI,
		pricing.NewCalculator(pricing.NewExchangeRateClient()),
	)

	err = queue.InitConsumer(&cfg.ConfigQueue, primitive.Prepare, func(d amqp091.Delivery) error {
		bookID := string(d.Body)

		err = preparationService.MapOnChunks(ctx, bookID, preparationService.GenerateChapterPreparation)
		if err != nil {
			if pricing.IsBudgetExceeded(err) {
				return book.RecordPermanentFailure(ctx, repo, bookID, primitive.Prepare, err)
			}
			return book.RecordFailure(ctx, repo, bookID, primitive.Prepare, err)
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
