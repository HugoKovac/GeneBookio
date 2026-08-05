package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"strings"

	"hkorpo/book/internal/book"
	"hkorpo/book/internal/platform/bucket"
	"hkorpo/book/internal/primitive"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

func main() {
	var (
		cfg      bucket.ConfigBucket
		ctx      = context.Background()
		epubPath = "petit_traite_de_manipulation_a_l_usage_des_honnetes_gens.epub"
	)

	if err := godotenv.Load("cmd/prepare_chapters/.env"); err != nil {
		log.Fatalf("load environment: %v", err)
	}

	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatal(err)
	}

	cClient, err := bucket.Init(&cfg)
	if err != nil {
		log.Fatal(err)
	}
	buckerRepo := book.NewBucketRepoImpl(cClient)
	bucketPath := strings.TrimSuffix(epubPath, ".epub")

	var builder strings.Builder
	iter := buckerRepo.GetFilesIteratorOfDir(ctx, primitive.ScriptsBucket, bucketPath+"/preparation/")
	for i := range iter {
		content, err := buckerRepo.GetBucketFileAsString(ctx, primitive.ScriptsBucket, i.Key)
		if err != nil {
			log.Fatal(err)
		}
		if _, err := builder.WriteString(content); err != nil {
			log.Fatal(err)
		}
	}
	result := builder.String()

	promptGenerateScript, err := buckerRepo.GetBucketFileAsString(ctx, primitive.PromptsBucket, primitive.NoneFictionGenerateScript)
	if err != nil {
		log.Fatal(err)
	}

	client := openai.NewClient()

	generated, err := client.Responses.New(ctx, responses.ResponseNewParams{
		Model: openai.ChatModelGPT5_2,
		Input: responses.ResponseNewParamsInputUnion{OfString: openai.String(promptGenerateScript + result)},
	})
	if err != nil {
		log.Fatalf("generate script: %v", err)
	}

	if err := buckerRepo.UploadStringAsTextFile(ctx, primitive.ScriptsBucket, fmt.Sprintf("%s/script.txt", bucketPath), generated.OutputText()); err != nil {
		log.Fatal(err)
	}
}
