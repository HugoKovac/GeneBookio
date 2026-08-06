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
	buckerRepo := book.NewBucketRepoImpl(cClient)

	err = queue.InitConsumer(&cfg.ConfigQueue, primitive.Prepare, func(d amqp091.Delivery) error {
		fileName := string(d.Body)

		promptPrepareChapter, err := buckerRepo.GetBucketFileAsString(ctx, primitive.PromptsBucket, primitive.NoneFictionPrepareChapter)
		if err != nil {
			return err
		}

		iter := buckerRepo.GetFilesIteratorOfDir(ctx, primitive.BooksBucket, "chunks/"+fileName+"/")

		client := openai.NewClient()
		count := 0
		for i := range iter {
			content, err := buckerRepo.GetBucketFileAsString(ctx, primitive.BooksBucket, i.Key)
			if err != nil {
				return err
			}

			preparation, err := client.Responses.New(ctx, responses.ResponseNewParams{
				Model: openai.ChatModelGPT5_2,
				Input: responses.ResponseNewParamsInputUnion{OfString: openai.String(
					promptPrepareChapter + content,
				)},
			})
			if err != nil {
				return errorwrapper.Wrap(fmt.Errorf("generate preparation: %v", err))
			}

			if err := buckerRepo.UploadStringAsTextFile(ctx, primitive.ScriptsBucket, fmt.Sprintf("%s/preparation/%d.txt", fileName, count), preparation.OutputText()); err != nil {
				return err
			}
			count++
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
