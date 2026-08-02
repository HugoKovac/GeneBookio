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

	if err := godotenv.Load("cmd/generate_script/.env"); err != nil && !errors.Is(err, os.ErrNotExist) {
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
	promptPrepareChapter, err := cClient.GetBucketFileAsString(ctx, primitive.PromptsBucket, primitive.NoneFictionPrepareChapter)
	if err != nil {
		log.Fatal(err)
	}

	iter := cClient.GetFilesIteratorOfDir(ctx, primitive.BooksBucket, "chunks/"+bucketPath+"/")

	client := openai.NewClient()
	count := 0
	for i := range iter {
		fmt.Println(i.Key)
		content, err := cClient.GetBucketFileAsString(ctx, primitive.BooksBucket, i.Key)
		if err != nil {
			log.Fatal(err)
		}

		preparation, err := client.Responses.New(ctx, responses.ResponseNewParams{
			Model: openai.ChatModelGPT5_2,
			Input: responses.ResponseNewParamsInputUnion{OfString: openai.String(
				promptPrepareChapter + content,
			)},
		})
		if err != nil {
			log.Fatalf("generate preparation: %v", err)
		}

		if err := cClient.UploadStringAsTextFile(ctx, primitive.ScriptsBucket, fmt.Sprintf("%s/preparation/%d.txt", bucketPath, count), preparation.OutputText()); err != nil {
			log.Fatal(err)
		}
		count++
	}
}
