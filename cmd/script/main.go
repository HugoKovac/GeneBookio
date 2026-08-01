package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
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

	pathSplit := strings.Split(epubPath, "/")
	bucketPath := strings.TrimSuffix(pathSplit[len(pathSplit)-1], ".epub")
	/*
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
			// fmt.Println(promptPrepareChapter)
			// fmt.Println(content)

			preparation, err := client.Responses.New(ctx, responses.ResponseNewParams{
				Model: openai.ChatModelGPT5_2,
				Input: responses.ResponseNewParamsInputUnion{OfString: openai.String(
					promptPrepareChapter + content,
				)},
			})
			if err != nil {
				log.Fatalf("generate preparation: %v", err)
			}

			if err := cClient.UploadStringAsTextFile(ctx, primitive.ScriptsBucket, fmt.Sprintf("preparation/%s/%d.txt", bucketPath, count), preparation.OutputText()); err != nil {
				log.Fatal(err)
			}
			count++
		}
	*/

	/////////////////////////////////////////

	var builder strings.Builder
	iter := cClient.GetFilesIteratorOfDir(ctx, primitive.ScriptsBucket, "preparation/"+bucketPath+"/")
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

func loadBookChunks(bookDirectory string) ([]string, error) {
	if bookDirectory == "" || filepath.Base(bookDirectory) != bookDirectory || bookDirectory == "." || bookDirectory == ".." {
		return nil, fmt.Errorf("invalid book directory %q", bookDirectory)
	}

	directory := filepath.Join("output", bookDirectory)
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("read book chunks from %q: %w", directory, err)
	}

	chunks := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}

		content, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read book chunk %q: %w", entry.Name(), err)
		}
		chunks = append(chunks, string(content))
	}

	if len(chunks) == 0 {
		return nil, fmt.Errorf("no book chunks found in %q", directory)
	}

	return chunks, nil
}
