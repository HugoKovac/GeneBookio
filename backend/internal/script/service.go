package script

import (
	"context"
	"fmt"
	"hkorpo/book/internal/book"
	pbucket "hkorpo/book/internal/platform/bucket"
	"hkorpo/book/internal/pricing"
	"hkorpo/book/internal/primitive"
	"strings"

	"github.com/google/uuid"
)

type AiAPI interface {
	// maxOutputTokens, when positive, caps the response length — see
	// BudgetAPI.CapOutputTokens.
	Request(ctx context.Context, request string, maxOutputTokens int64) (string, primitive.ModelUsage, error)
	ModelName() string
}

// BudgetAPI checks a stage's AI usage cost against a EUR budget.
type BudgetAPI interface {
	// CheckBudget is the post-call check: did this call's real (reported)
	// usage exceed the budget?
	CheckBudget(ctx context.Context, stage, model string, usage primitive.ModelUsage, limitEUR float64) error
	// CapOutputTokens is the pre-call check: given an estimated input size,
	// how many output tokens can the remaining budget still afford? It
	// returns an error — before any request is made — if the (estimated)
	// input alone already exceeds the budget.
	CapOutputTokens(ctx context.Context, stage, model string, inputTokens int64, limitEUR float64) (int64, error)
}

// budgetEUR is the maximum a single book may spend on script generation
// before the book fails permanently (see GenerateScript).
const budgetEUR = 1.0

// Service merges a book's prepared chapter chunks into one narration script.
type Service struct {
	repo       book.Repository
	bucketRepo book.BucketRepo
	queueRepo  book.QueueRepo
	aiAPI      AiAPI
	budgetAPI  BudgetAPI
}

func NewService(repo book.Repository, bucketRepo book.BucketRepo, queueRepo book.QueueRepo, aiAPI AiAPI, budgetAPI BudgetAPI) *Service {
	return &Service{repo: repo, bucketRepo: bucketRepo, queueRepo: queueRepo, aiAPI: aiAPI, budgetAPI: budgetAPI}
}

func (s *Service) GenerateScript(ctx context.Context, bookID string) error {
	b, err := s.repo.GetBookByID(ctx, uuid.MustParse(bookID))
	if err != nil {
		return err
	}

	var builder strings.Builder
	if b.Language == primitive.English {
		fmt.Fprintf(&builder, "Title: %s\nAuthor(s): %s\nDescription: %s\n\n", b.Title, strings.Join(b.AuthorNames, ", "), b.Description)
	} else {
		fmt.Fprintf(&builder, "Titre : %s\nAuteur(s) : %s\nDescription : %s\n\n", b.Title, strings.Join(b.AuthorNames, ", "), b.Description)
	}

	iter := s.bucketRepo.GetFilesIteratorOfDir(ctx, primitive.ScriptsBucket, bookID+"/preparation/")
	for i := range iter {
		content, err := s.bucketRepo.GetBucketFileAsString(ctx, primitive.ScriptsBucket, i.Key)
		if err != nil {
			return err
		}
		if _, err := builder.WriteString(content); err != nil {
			return err
		}
	}
	result := builder.String()

	promptGenerateScript, err := s.bucketRepo.GetBucketFileAsString(ctx, primitive.PromptsBucket, primitive.PromptFile(primitive.GenerateScriptPromptKind(b.Genre), b.Language))
	if err != nil {
		return err
	}

	request := promptGenerateScript + result

	maxOutputTokens, err := s.budgetAPI.CapOutputTokens(ctx, primitive.GenerateScript, s.aiAPI.ModelName(), pricing.EstimateTokens(request), budgetEUR)
	if err != nil {
		return err
	}

	output, usage, err := s.aiAPI.Request(ctx, request, maxOutputTokens)
	if err != nil {
		return err
	}

	if err := s.bucketRepo.UploadString(ctx, primitive.ScriptsBucket, fmt.Sprintf("%s/script.txt", bookID), output, pbucket.TEXT); err != nil {
		return err
	}

	if err := s.repo.AddTokenUsage(ctx, uuid.MustParse(bookID), s.aiAPI.ModelName(), usage); err != nil {
		return err
	}

	if err := s.budgetAPI.CheckBudget(ctx, primitive.GenerateScript, s.aiAPI.ModelName(), usage, budgetEUR); err != nil {
		return err
	}

	err = s.repo.UpdateBookStage(ctx, uuid.MustParse(bookID), book.ScriptGenerated)
	if err != nil {
		return err
	}

	return s.queueRepo.PostMessage(bookID)
}
