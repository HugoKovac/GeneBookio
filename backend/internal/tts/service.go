package tts

import (
	"context"
	"fmt"
	"hkorpo/book/internal/book"
	pbucket "hkorpo/book/internal/platform/bucket"
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

	// Unlike preparation/script, TTS is priced purely per input character
	// (see OpenAiTTSClient) and the full input is already in hand — so this
	// pre-call check is exact, not an estimate, and fully prevents the
	// spend rather than just detecting it afterwards.
	estimatedUsage := primitive.ModelUsage{InputTokens: int64(len([]rune(bookContent)))}
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
