package main

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"

	"hkorpo/book/internal/book"
	"hkorpo/book/internal/platform/bucket"
	"hkorpo/book/internal/primitive"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

func main() {
	epubPath := "petit_traite_de_manipulation_a_l_usage_des_honnetes_gens.epub"

	if err := godotenv.Load("cmd/script/.env"); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("load environment: %v", err)
	}

	var (
		cfg bucket.ConfigBucket
		ctx = context.Background()
	)

	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatal(err)
	}

	cClient, err := bucket.Init(ctx, &cfg)
	if err != nil {
		log.Fatal(err)
	}

	bookContent, err := cClient.GetBucketFileAsBytes(ctx, primitive.BooksBucket, epubPath)
	if err != nil {
		log.Fatal(err)
	}

	chunks, err := book.ExtractEPUB(bookContent)
	if err != nil {
		log.Fatal(err)
	}

	pathSplit := strings.Split(epubPath, "/")
	bucketPath := strings.TrimSuffix(pathSplit[len(pathSplit)-1], ".epub")
	for name, content := range chunks {
		if err := cClient.UploadStringAsTextFile(ctx, primitive.BooksBucket, "chunks/"+bucketPath+"/"+name, content); err != nil {
			log.Fatal(err)
		}
	}
}
