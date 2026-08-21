package tts

import (
	"context"
	"fmt"
	"hkorpo/book/internal/book"
	pbucket "hkorpo/book/internal/platform/bucket"
	"hkorpo/book/internal/pricing"
	"hkorpo/book/internal/primitive"
	"io"

	"github.com/google/uuid"
)

type TTSAPI interface {
	CreateAudioFromString(ctx context.Context, content string, language primitive.Language) (io.ReadCloser, int64, primitive.ModelUsage, error)
	ModelName() string
}

// BudgetAPI checks a stage's AI usage cost against a EUR budget.
type BudgetAPI interface {
	CheckBudget(ctx context.Context, stage, model string, usage primitive.ModelUsage, limitEUR float64) error
}

// budgetEUR is the maximum a single book may spend on TTS synthesis before
// the book fails permanently (see CreateAudioFromScript).
const budgetEUR = 2.0

// Service synthesizes a book's narration script into audio.
type Service struct {
	repo       book.Repository
	bucketRepo book.BucketRepo
	ttsAPI     TTSAPI
	budgetAPI  BudgetAPI
}

func NewService(repo book.Repository, bucketRepo book.BucketRepo, ttsAPI TTSAPI, budgetAPI BudgetAPI) *Service {
	return &Service{repo: repo, bucketRepo: bucketRepo, ttsAPI: ttsAPI, budgetAPI: budgetAPI}
}

func (s *Service) CreateAudioFromScript(ctx context.Context, bookID string) error {
	b, err := s.repo.GetBookByID(ctx, uuid.MustParse(bookID))
	if err != nil {
		return err
	}

	bookContent, err := s.bucketRepo.GetBucketFileAsString(ctx, primitive.ScriptsBucket, fmt.Sprintf("%s/script.txt", bookID))
	if err != nil {
		return err
	}

	// Pre-call check on the input side: for a character-priced model
	// (tts-1/tts-1-hd) this is exact, since the full input is already in
	// hand, and fully prevents the spend rather than just detecting it
	// afterwards. For a token-priced model (gpt-4o-mini-tts) the dominant
	// cost is the output (audio) side, which isn't known until the API
	// reports real usage — see pricing.EstimateTTSInputUsage and the
	// post-call CheckBudget below, which is what catches that.
	estimatedUsage := pricing.EstimateTTSInputUsage(s.ttsAPI.ModelName(), bookContent)
	if err := s.budgetAPI.CheckBudget(ctx, primitive.GenerateTTS, s.ttsAPI.ModelName(), estimatedUsage, budgetEUR); err != nil {
		return err
	}

	audio, len, usage, err := s.ttsAPI.CreateAudioFromString(ctx, bookContent, b.Language)
	if err != nil {
		return err
	}
	if err := s.bucketRepo.UploadReader(ctx, primitive.AudioBucket, bookID, audio, len, pbucket.WAV); err != nil {
		return err
	}
	if err := s.repo.AddTokenUsage(ctx, uuid.MustParse(bookID), s.ttsAPI.ModelName(), usage); err != nil {
		return err
	}
	if err := s.budgetAPI.CheckBudget(ctx, primitive.GenerateTTS, s.ttsAPI.ModelName(), usage, budgetEUR); err != nil {
		return err
	}
	return s.repo.UpdateBookStage(ctx, uuid.MustParse(bookID), book.TTSGenerated)
}
