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

	if err := godotenv.Load("cmd/generate_script/.env"); err != nil {
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
	promptPrepareChapter, err := buckerRepo.GetBucketFileAsString(ctx, primitive.PromptsBucket, primitive.NoneFictionPrepareChapter)
	if err != nil {
		log.Fatal(err)
	}

	iter := buckerRepo.GetFilesIteratorOfDir(ctx, primitive.BooksBucket, "chunks/"+bucketPath+"/")

	client := openai.NewClient()
	count := 0
	for i := range iter {
		fmt.Println(i.Key)
		content, err := buckerRepo.GetBucketFileAsString(ctx, primitive.BooksBucket, i.Key)
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

		if err := buckerRepo.UploadStringAsTextFile(ctx, primitive.ScriptsBucket, fmt.Sprintf("%s/preparation/%d.txt", bucketPath, count), preparation.OutputText()); err != nil {
			log.Fatal(err)
		}
		count++
	}
}
