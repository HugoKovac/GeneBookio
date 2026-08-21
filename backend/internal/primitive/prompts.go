package primitive

type PromptKind string

const (
	FictionPrepareChapter     PromptKind = "fiction-prepare-chapter"
	FictionGenerateScript     PromptKind = "fiction-generate-script"
	NoneFictionPrepareChapter PromptKind = "none-fiction-prepare-chapter"
	NoneFictionGenerateScript PromptKind = "none-fiction-generate-script"
)

// PromptFile returns the prompts-bucket object key for a prompt kind in a
// given language, e.g. "none-fiction-generate-script.en.md".
func PromptFile(kind PromptKind, language Language) string {
	return string(kind) + "." + language.String() + ".md"
}

// PrepareChapterPromptKind and GenerateScriptPromptKind pick the fiction vs.
// none-fiction variant of a stage's prompt based on the book's genre.
func PrepareChapterPromptKind(genre Genre) PromptKind {
	if genre == Fiction {
		return FictionPrepareChapter
	}
	return NoneFictionPrepareChapter
}

func GenerateScriptPromptKind(genre Genre) PromptKind {
	if genre == Fiction {
		return FictionGenerateScript
	}
	return NoneFictionGenerateScript
}
