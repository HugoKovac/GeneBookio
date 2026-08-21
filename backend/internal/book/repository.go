package book

import (
	"context"
	"hkorpo/book/internal/primitive"
	"hkorpo/book/pkg/ent"
	"hkorpo/book/pkg/ent/book"
	"hkorpo/book/pkg/errorwrapper"

	"github.com/google/uuid"
)

type Repository interface {
	CreateBook(ctx context.Context, book *Book) (*Book, error)
	UpdateBookStage(ctx context.Context, bookID uuid.UUID, s Stage) error
	MarkBookFailed(ctx context.Context, bookID uuid.UUID, stage, message string) error
	// MarkBookFailedPermanently is like MarkBookFailed but also sets
	// RetryDisabled, so catalog.Service.RetryFailedStage refuses to retry it.
	MarkBookFailedPermanently(ctx context.Context, bookID uuid.UUID, stage, message string) error
	ClearBookFailure(ctx context.Context, bookID uuid.UUID) error
	GetSavedBookByKey(ctx context.Context, bookKey string) (*Book, error)
	GetBookByID(ctx context.Context, bookID uuid.UUID) (*Book, error)
	GetBooks(ctx context.Context, page, limit int) ([]*Book, error)
	// AddTokenUsage accumulates usage onto whatever is already recorded for
	// model on this book (read-modify-write). Safe as long as at most one
	// pipeline stage writes to a given book at a time, which holds today:
	// a book moves through stages sequentially, and preparation's per-chunk
	// AI calls are aggregated in memory before a single call here.
	AddTokenUsage(ctx context.Context, bookID uuid.UUID, model string, usage primitive.ModelUsage) error
}

type RepositoryImpl struct {
	dbClient *ent.Client
}

func NewRepositoryImpl(dbClient *ent.Client) *RepositoryImpl {
	return &RepositoryImpl{
		dbClient: dbClient,
	}
}

func (r *RepositoryImpl) CreateBook(ctx context.Context, book *Book) (*Book, error) {
	query := r.dbClient.Book.Create().
		SetTitle(book.Title).
		SetKey(book.Key).
		SetCoverURL(book.CoverURL).
		SetAuthorNames(book.AuthorNames).
		SetDescription(book.Description)

	if book.Language != "" {
		query.SetLanguage(book.Language)
	}
	if book.Genre != "" {
		query.SetGenre(book.Genre)
	}

	e, err := query.Save(ctx)
	if err != nil {
		return nil, errorwrapper.Wrap(err)
	}
	return &Book{
		ID:          e.ID,
		Title:       e.Title,
		AuthorNames: e.AuthorNames,
		CoverURL:    e.CoverURL,
		Key:         e.Key,
		Description: e.Description,
		Language:    e.Language,
		Genre:       e.Genre,
	}, nil
}

func (r *RepositoryImpl) UpdateBookStage(ctx context.Context, bookID uuid.UUID, s Stage) error {
	query := r.dbClient.Book.UpdateOneID(bookID)

	switch s {
	case Uploaded:
		query.SetUploaded(true)
	case Parsed:
		query.SetParsed(true)
	case Prepared:
		query.SetPrepared(true)
	case ScriptGenerated:
		query.SetScriptGenerated(true)
	case TTSGenerated:
		query.SetTtsGenerated(true)
	}

	return query.Exec(ctx)
}

func (r *RepositoryImpl) MarkBookFailed(ctx context.Context, bookID uuid.UUID, stage, message string) error {
	return errorwrapper.Wrap(r.dbClient.Book.UpdateOneID(bookID).
		SetFailed(true).
		SetFailedStage(stage).
		SetErrorMessage(message).
		Exec(ctx))
}

func (r *RepositoryImpl) MarkBookFailedPermanently(ctx context.Context, bookID uuid.UUID, stage, message string) error {
	return errorwrapper.Wrap(r.dbClient.Book.UpdateOneID(bookID).
		SetFailed(true).
		SetFailedStage(stage).
		SetErrorMessage(message).
		SetRetryDisabled(true).
		Exec(ctx))
}

func (r *RepositoryImpl) ClearBookFailure(ctx context.Context, bookID uuid.UUID) error {
	return errorwrapper.Wrap(r.dbClient.Book.UpdateOneID(bookID).
		SetFailed(false).
		ClearFailedStage().
		ClearErrorMessage().
		SetRetryDisabled(false).
		Exec(ctx))
}

func (r *RepositoryImpl) AddTokenUsage(ctx context.Context, bookID uuid.UUID, model string, usage primitive.ModelUsage) error {
	e, err := r.dbClient.Book.Get(ctx, bookID)
	if err != nil {
		return errorwrapper.Wrap(err)
	}
	updated := primitive.TokenUsage(e.TokenUsage).Add(model, usage)
	return errorwrapper.Wrap(r.dbClient.Book.UpdateOneID(bookID).SetTokenUsage(updated).Exec(ctx))
}

func (r *RepositoryImpl) GetSavedBookByKey(ctx context.Context, bookKey string) (*Book, error) {
	e, err := r.dbClient.Book.Query().Where(
		book.KeyEQ(bookKey),
	).First(ctx)
	if err != nil {
		return nil, errorwrapper.Wrap(err)
	}
	return fromEntBook(e), nil
}

func (r *RepositoryImpl) GetBookByID(ctx context.Context, bookID uuid.UUID) (*Book, error) {
	e, err := r.dbClient.Book.Get(ctx, bookID)
	if err != nil {
		return nil, errorwrapper.Wrap(err)
	}
	return fromEntBook(e), nil
}

func (r *RepositoryImpl) GetBooks(ctx context.Context, page, limit int) ([]*Book, error) {
	e, err := r.dbClient.Book.Query().
		Where().
		Order(ent.Desc(book.FieldCreatedAt)).
		Offset(page * limit).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, errorwrapper.Wrap(err)
	}
	rtn := make([]*Book, 0, len(e))
	for _, b := range e {
		rtn = append(rtn, fromEntBook(b))
	}
	return rtn, nil
}

func fromEntBook(e *ent.Book) *Book {
	return &Book{
		ID:              e.ID,
		Title:           e.Title,
		AuthorNames:     e.AuthorNames,
		CoverURL:        e.CoverURL,
		Key:             e.Key,
		Description:     e.Description,
		Language:        e.Language,
		Genre:           e.Genre,
		Uploaded:        e.Uploaded,
		Parsed:          e.Parsed,
		Prepared:        e.Prepared,
		ScriptGenerated: e.ScriptGenerated,
		TTSGenerated:    e.TtsGenerated,
		Failed:          e.Failed,
		FailedStage:     e.FailedStage,
		ErrorMessage:    e.ErrorMessage,
		RetryDisabled:   e.RetryDisabled,
		TokenUsage:      e.TokenUsage,
	}
}
