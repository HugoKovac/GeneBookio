package script

import (
	"context"
	"fmt"
	"hkorpo/book/internal/book"
	pbucket "hkorpo/book/internal/platform/bucket"
	"hkorpo/book/internal/primitive"
	"strings"

	"github.com/google/uuid"
)

type AiAPI interface {
	Request(ctx context.Context, request string) (string, error)
}

// Service merges a book's prepared chapter chunks into one narration script.
type Service struct {
	repo       book.Repository
	bucketRepo book.BucketRepo
	queueRepo  book.QueueRepo
	aiAPI      AiAPI
}

func NewService(repo book.Repository, bucketRepo book.BucketRepo, queueRepo book.QueueRepo, aiAPI AiAPI) *Service {
	return &Service{repo: repo, bucketRepo: bucketRepo, queueRepo: queueRepo, aiAPI: aiAPI}
}

func (s *Service) GenerateScript(ctx context.Context, bookID string) error {
	var builder strings.Builder
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

	promptGenerateScript, err := s.bucketRepo.GetBucketFileAsString(ctx, primitive.PromptsBucket, primitive.NoneFictionGenerateScript)
	if err != nil {
		return err
	}

	output, err := s.aiAPI.Request(ctx, promptGenerateScript+result)
	if err != nil {
		return err
	}

	if err := s.bucketRepo.UploadString(ctx, primitive.ScriptsBucket, fmt.Sprintf("%s/script.txt", bookID), output, pbucket.TEXT); err != nil {
		return err
	}

	err = s.repo.UpdateBookStage(ctx, uuid.MustParse(bookID), book.ScriptGenerated)
	if err != nil {
		return err
	}

	return s.queueRepo.PostMessage(bookID)
}
