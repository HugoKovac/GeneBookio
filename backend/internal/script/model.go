package script

type Script struct {
	PreparationPrompt string
	GenerationPrompt  string
	Content           string
	BookChunks        []string
}
