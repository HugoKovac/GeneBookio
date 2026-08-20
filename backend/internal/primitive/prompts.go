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
