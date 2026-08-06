package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"hkorpo/book/internal/book"
	"hkorpo/book/internal/platform/bucket"
	"hkorpo/book/internal/platform/queue"
	"hkorpo/book/internal/primitive"
	"hkorpo/book/pkg/errorpkg"
	"hkorpo/book/pkg/errorwrapper"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
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

	if err := godotenv.Load("cmd/generate_script/.env"); err != nil {
		errorpkg.ExitTrace(fmt.Errorf("load environment: %v", err))
	}

	if err := envconfig.Process("", &cfg); err != nil {
		errorpkg.ExitTrace(err)
	}

	cClient, err := bucket.Init(&cfg.ConfigBucket)
	if err != nil {
		errorpkg.ExitTrace(err)
	}
	buckerRepo := book.NewBucketRepoImpl(cClient)

	err = queue.InitConsumer(&cfg.ConfigQueue, primitive.Generate, func(d amqp091.Delivery) error {
		fileName := string(d.Body)
		var builder strings.Builder
		iter := buckerRepo.GetFilesIteratorOfDir(ctx, primitive.ScriptsBucket, fileName+"/preparation/")
		for i := range iter {
			content, err := buckerRepo.GetBucketFileAsString(ctx, primitive.ScriptsBucket, i.Key)
			if err != nil {
				return err
			}
			if _, err := builder.WriteString(content); err != nil {
				return err
			}
		}
		result := builder.String()

		promptGenerateScript, err := buckerRepo.GetBucketFileAsString(ctx, primitive.PromptsBucket, primitive.NoneFictionGenerateScript)
		if err != nil {
			return err
		}

		client := openai.NewClient()

		generated, err := client.Responses.New(ctx, responses.ResponseNewParams{
			Model: openai.ChatModelGPT5_2,
			Input: responses.ResponseNewParamsInputUnion{OfString: openai.String(promptGenerateScript + result)},
		})
		if err != nil {
			return errorwrapper.Wrap(fmt.Errorf("generate script: %v", err))
		}

		if err := buckerRepo.UploadStringAsTextFile(ctx, primitive.ScriptsBucket, fmt.Sprintf("%s/script.txt", fileName), generated.OutputText()); err != nil {
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
