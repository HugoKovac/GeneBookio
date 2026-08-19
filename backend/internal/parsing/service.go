package parsing

import (
	"context"
	"hkorpo/book/internal/book"
	pbucket "hkorpo/book/internal/platform/bucket"
	"hkorpo/book/internal/primitive"
	"log"

	"github.com/google/uuid"
)

type EpubParser interface {
	ExtractEPUB(epubContent []byte) (map[string]string, error)
}

// Service splits an uploaded EPUB into per-chapter text chunks.
type Service struct {
	repo       book.Repository
	bucketRepo book.BucketRepo
	queueRepo  book.QueueRepo
	epubParser EpubParser
}

func NewService(repo book.Repository, bucketRepo book.BucketRepo, queueRepo book.QueueRepo, epubParser EpubParser) *Service {
	return &Service{repo: repo, bucketRepo: bucketRepo, queueRepo: queueRepo, epubParser: epubParser}
}

func (s *Service) GetBookAsChunks(ctx context.Context, bookID string) (map[string]string, error) {
	bookContent, err := s.bucketRepo.GetBucketFileAsBytes(ctx, primitive.BooksBucket, "uploads/"+bookID)
	if err != nil {
		return nil, err
	}
	return s.epubParser.ExtractEPUB(bookContent)
}

func (s *Service) UploadBookChunks(ctx context.Context, bookID string, chunks map[string]string) error {
	for name, content := range chunks {
		if err := s.bucketRepo.UploadString(ctx, primitive.BooksBucket, "chunks/"+bookID+"/"+name, content, pbucket.TEXT); err != nil {
			log.Fatal(err)
		}
	}
	if err := s.repo.UpdateBookStage(ctx, uuid.MustParse(bookID), book.Parsed); err != nil {
		return err
	}
	if err := s.queueRepo.PostMessage(bookID); err != nil {
		return err
	}
	return nil
}
