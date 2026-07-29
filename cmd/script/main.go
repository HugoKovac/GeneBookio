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

	"hkorpo/book/internal/script"

	"github.com/joho/godotenv"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

//go:embed prompt1.md
var preparationPrompt string

//go:embed prompt2.md
var generationPrompt string

func main() {
	if len(os.Args) != 2 {
		log.Fatal("usage: go run ./cmd/script <book-directory>")
	}

	if err := godotenv.Load("cmd/script/.env"); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("load environment: %v", err)
	}

	bookChunks, err := loadBookChunks(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	workflow := script.Script{
		PreparationPrompt: preparationPrompt,
		GenerationPrompt:  generationPrompt,
		BookChunks:        bookChunks,
	}

	ctx := context.Background()
	client := openai.NewClient()

	preparation, err := client.Responses.New(ctx, responses.ResponseNewParams{
		Model: openai.ChatModelGPT5_2,
		Input: responses.ResponseNewParamsInputUnion{OfString: openai.String(
			workflow.PreparationPrompt + "\n\n--- DOCUMENT À ANALYSER ---\n" + strings.Join(workflow.BookChunks, "\n\n"),
		)},
	})
	if err != nil {
		log.Fatalf("generate preparation: %v", err)
	}

	generated, err := client.Responses.New(ctx, responses.ResponseNewParams{
		Model:              openai.ChatModelGPT5_2,
		PreviousResponseID: openai.String(preparation.ID),
		Input:              responses.ResponseNewParamsInputUnion{OfString: openai.String(workflow.GenerationPrompt)},
	})
	if err != nil {
		log.Fatalf("generate script: %v", err)
	}

	workflow.Content = generated.OutputText()
	fmt.Print(workflow.Content)
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
