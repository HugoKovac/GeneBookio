package book

import (
	"context"
	"hkorpo/book/pkg/ent"
	"hkorpo/book/pkg/ent/book"
	"hkorpo/book/pkg/errorwrapper"

	"github.com/google/uuid"
)

type RepositoryImpl struct {
	dbClient *ent.Client
}

func NewRepositoryImpl(dbClient *ent.Client) *RepositoryImpl {
	return &RepositoryImpl{
		dbClient: dbClient,
	}
}

func (r *RepositoryImpl) CreateBook(ctx context.Context, book *Book) (*Book, error) {
	e, err := r.dbClient.Book.Create().
		SetTitle(book.Title).
		SetKey(book.Key).
		SetCoverURL(book.CoverURL).
		SetAuthorKeys(book.AuthorKeys).
		SetAuthorNames(book.AuthorNames).
		SetDescription(book.Description).
		Save(ctx)
	if err != nil {
		return nil, errorwrapper.Wrap(err)
	}
	return &Book{
		ID:          e.ID,
		Title:       e.Title,
		AuthorNames: e.AuthorNames,
		CoverURL:    e.CoverURL,
		Key:         e.Key,
		AuthorKeys:  e.AuthorKeys,
		Description: e.Description,
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
	}

	return query.Exec(ctx)
}

func (r *RepositoryImpl) GetSavedBookByKey(ctx context.Context, bookKey string) (*Book, error) {
	e, err := r.dbClient.Book.Query().Where(
		book.KeyEQ(bookKey),
	).First(ctx)
	if err != nil {
		return nil, errorwrapper.Wrap(err)
	}
	return &Book{
		ID:              e.ID,
		Title:           e.Title,
		AuthorNames:     e.AuthorNames,
		CoverURL:        e.CoverURL,
		Key:             e.Key,
		AuthorKeys:      e.AuthorKeys,
		Description:     e.Description,
		Uploaded:        e.Uploaded,
		Parsed:          e.Parsed,
		Prepared:        e.Prepared,
		ScriptGenerated: e.ScriptGenerated,
	}, nil
}
