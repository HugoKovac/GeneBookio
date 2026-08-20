package book

import (
	"hkorpo/book/internal/primitive"

	"github.com/google/uuid"
)

type Book struct {
	ID                                          uuid.UUID
	Title                                       string
	AuthorNames                                 []string
	CoverURL                                    string
	Key                                         string
	AuthorKeys                                  []string
	Description                                 string
	Language                                    primitive.Language
	Uploaded, Parsed, Prepared, ScriptGenerated bool
}

type QueryURI struct {
	Query string `uri:"query" validate:"required"`
}

type BookDTO struct {
	Title       string   `json:"title"`
	Authors     []string `json:"authors,omitempty"`
	AuthorNames []string `json:"author_names,omitempty"`
	CoverURL    string   `json:"cover_url"`
	Key         string   `json:"key"`
	Descriptiom string   `json:"description"`
}

type Stage int

const (
	Uploaded Stage = iota
	Parsed
	Prepared
	ScriptGenerated
	TTSGenerated
)
