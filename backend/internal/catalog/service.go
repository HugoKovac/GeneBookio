package catalog

import (
	"context"
	"fmt"
	"hkorpo/book/internal/book"
	"hkorpo/book/internal/primitive"
	"hkorpo/book/pkg/errorwrapper"
	"io"

	"github.com/google/uuid"
)

// PricingAPI turns a book's recorded TokenUsage into a display cost.
// CostEUR returning an error just means no exchange rate is available right
// now (see pricing.ExchangeRateClient) — GetBooks treats that as "omit the
// EUR figure", not as a request failure.
type PricingAPI interface {
	CostUSD(usage primitive.TokenUsage) float64
	CostEUR(ctx context.Context, usdCost float64) (float64, error)
}

// Service manages saved book records and their generated audio.
type Service struct {
	repo       book.Repository
	bucketRepo book.BucketRepo
	// queueRepos publishes a retry message onto the queue for a given failed
	// stage (keyed by primitive.QueueChannel, e.g. "prepare"), so a failed
	// book can resume from wherever it stopped instead of from the start.
	queueRepos map[string]book.QueueRepo
	pricingAPI PricingAPI
}

func NewService(repo book.Repository, bucketRepo book.BucketRepo, queueRepos map[string]book.QueueRepo, pricingAPI PricingAPI) *Service {
	return &Service{repo: repo, bucketRepo: bucketRepo, queueRepos: queueRepos, pricingAPI: pricingAPI}
}

func (s *Service) SaveBook(ctx context.Context, b *book.Book) (*book.Book, error) {
	return s.repo.CreateBook(ctx, b)
}

func (s *Service) GetSavedBookByKey(ctx context.Context, bookKey string) (*book.Book, error) {
	return s.repo.GetSavedBookByKey(ctx, bookKey)
}

// BookWithCost is a saved book plus its display cost, computed from the
// book's TokenUsage. CostEUR is omitted when no exchange rate is available.
type BookWithCost struct {
	*book.Book
	CostUSD float64  `json:"CostUSD"`
	CostEUR *float64 `json:"CostEUR,omitempty"`
}

func (s *Service) GetBooks(ctx context.Context, page, limit int) ([]*BookWithCost, error) {
	books, err := s.repo.GetBooks(ctx, page, limit)
	if err != nil {
		return nil, err
	}

	result := make([]*BookWithCost, 0, len(books))
	for _, b := range books {
		entry := &BookWithCost{Book: b, CostUSD: s.pricingAPI.CostUSD(b.TokenUsage)}
		if eur, err := s.pricingAPI.CostEUR(ctx, entry.CostUSD); err == nil {
			entry.CostEUR = &eur
		}
		result = append(result, entry)
	}
	return result, nil
}

func (s *Service) GetBucketObjectAsReader(ctx context.Context, bucket primitive.Bucket, path string) (io.Reader, int64, string, error) {
	return s.bucketRepo.GetBucketObjectAsReader(ctx, bucket, path)
}

// RetryFailedStage re-publishes bookID onto the queue for the stage it last
// failed at, so the pipeline resumes from there instead of from the start.
func (s *Service) RetryFailedStage(ctx context.Context, bookID string) error {
	id, err := uuid.Parse(bookID)
	if err != nil {
		return errorwrapper.Wrap(err)
	}

	b, err := s.repo.GetBookByID(ctx, id)
	if err != nil {
		return err
	}

	if !b.Failed {
		return errorwrapper.Wrap(fmt.Errorf("book %s has no failed stage to retry", bookID))
	}

	if b.RetryDisabled {
		return errorwrapper.Wrap(fmt.Errorf("book %s failed permanently and cannot be retried: %s", bookID, b.ErrorMessage))
	}

	queueRepo, ok := s.queueRepos[b.FailedStage]
	if !ok {
		return errorwrapper.Wrap(fmt.Errorf("unknown failed stage %q", b.FailedStage))
	}

	if err := queueRepo.PostMessage(bookID); err != nil {
		return err
	}

	return s.repo.ClearBookFailure(ctx, id)
}
