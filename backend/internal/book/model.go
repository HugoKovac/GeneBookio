package book

import (
	"hkorpo/book/internal/primitive"

	"github.com/google/uuid"
)

type Book struct {
	ID                                                        uuid.UUID
	Title                                                     string
	AuthorNames                                               []string
	CoverURL                                                  string
	Key                                                       string
	Description                                               string
	Language                                                  primitive.Language
	Genre                                                     primitive.Genre
	Uploaded, Parsed, Prepared, ScriptGenerated, TTSGenerated bool
	Failed                                                    bool
	FailedStage                                               string
	ErrorMessage                                              string
	// RetryDisabled marks Failed as permanent — see book.RecordPermanentFailure.
	RetryDisabled bool
	TokenUsage    primitive.TokenUsage
}

type QueryURI struct {
	Query string `uri:"query" validate:"required"`
}

type BookDTO struct {
	Title       string   `json:"title"`
	AuthorNames []string `json:"author_names,omitempty"`
	CoverURL    string   `json:"cover_url"`
	Key         string   `json:"key"`
	Description string   `json:"description"`
}

type Stage int

const (
	Uploaded Stage = iota
	Parsed
	Prepared
	ScriptGenerated
	TTSGenerated
)
