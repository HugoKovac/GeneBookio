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

// Service synthesizes a book's narration script into audio.
type Service struct {
	repo       book.Repository
	bucketRepo book.BucketRepo
	ttsAPI     TTSAPI
}

func NewService(repo book.Repository, bucketRepo book.BucketRepo, ttsAPI TTSAPI) *Service {
	return &Service{repo: repo, bucketRepo: bucketRepo, ttsAPI: ttsAPI}
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
	return s.repo.UpdateBookStage(ctx, uuid.MustParse(bookID), book.TTSGenerated)
}
