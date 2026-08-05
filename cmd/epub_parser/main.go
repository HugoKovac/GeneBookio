package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"hkorpo/book/internal/book"
	"hkorpo/book/internal/platform/bucket"
	"hkorpo/book/internal/platform/queue"
	"hkorpo/book/internal/primitive"

	"github.com/joho/godotenv"
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

	if err := godotenv.Load("cmd/epub_parser/.env"); err != nil {
		log.Fatalf("load environment: %v", err)
	}

	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatal(err)
	}

	cClient, err := bucket.Init(&cfg.ConfigBucket)
	if err != nil {
		log.Fatal(err)
	}

	buckerRepo := book.NewBucketRepoImpl(cClient)

	err = queue.InitConsumer(&cfg.ConfigQueue, primitive.Uploads, func(d amqp091.Delivery) {
		bookName := string(d.Body)
		bookContent, err := buckerRepo.GetBucketFileAsBytes(ctx, primitive.BooksBucket, bookName)
		if err != nil {
			log.Fatal(err)
		}

		chunks, err := book.ExtractEPUB(bookContent)
		if err != nil {
			log.Fatal(err)
		}

		bucketPath := strings.TrimSuffix(bookName, ".epub")
		for name, content := range chunks {
			if err := buckerRepo.UploadStringAsTextFile(ctx, primitive.BooksBucket, "chunks/"+bucketPath+"/"+name, content); err != nil {
				log.Fatal(err)
			}
		}
	})
	if err != nil {
		log.Fatal(err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan

}
