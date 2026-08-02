package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"hkorpo/book/internal/platform/bucket"
	"hkorpo/book/internal/primitive"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

func main() {
	epubPath := "petit_traite_de_manipulation_a_l_usage_des_honnetes_gens.epub"

	if err := godotenv.Load("cmd/prepare_chapters/.env"); err != nil && !errors.Is(err, os.ErrNotExist) {
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

	pathSplit := strings.Split(epubPath, "/")
	bucketPath := strings.TrimSuffix(pathSplit[len(pathSplit)-1], ".epub")

	var builder strings.Builder
	iter := cClient.GetFilesIteratorOfDir(ctx, primitive.ScriptsBucket, bucketPath+"/preparation/")
	for i := range iter {
		content, err := cClient.GetBucketFileAsString(ctx, primitive.ScriptsBucket, i.Key)
		if err != nil {
			log.Fatal(err)
		}
		if _, err := builder.WriteString(content); err != nil {
			log.Fatal(err)
		}
	}
	result := builder.String()

	promptGenerateScript, err := cClient.GetBucketFileAsString(ctx, primitive.PromptsBucket, primitive.NoneFictionGenerateScript)
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

	if err := cClient.UploadStringAsTextFile(ctx, primitive.ScriptsBucket, fmt.Sprintf("%s/script.txt", bucketPath), generated.OutputText()); err != nil {
		log.Fatal(err)
	}
}
