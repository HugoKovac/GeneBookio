package primitive

// ModelUsage accumulates the usage recorded for a single AI model, across
// however many calls to it a pipeline stage made for one book. For models
// priced per input character rather than per token (e.g. tts-1), InputTokens
// holds the character count instead — the pricing math only cares about the
// per-unit count, not what the unit is called.
type ModelUsage struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
	TotalTokens  int64 `json:"total_tokens"`
}

func (m ModelUsage) Add(other ModelUsage) ModelUsage {
	return ModelUsage{
		InputTokens:  m.InputTokens + other.InputTokens,
		OutputTokens: m.OutputTokens + other.OutputTokens,
		TotalTokens:  m.TotalTokens + other.TotalTokens,
	}
}

// TokenUsage maps a model name (e.g. "gpt-5-mini", "tts-1") to the usage
// accumulated for one book across every pipeline stage that called it.
type TokenUsage map[string]ModelUsage

// Add merges usage for model into u, returning the updated map so it can be
// used on a nil TokenUsage.
func (u TokenUsage) Add(model string, usage ModelUsage) TokenUsage {
	if u == nil {
		u = TokenUsage{}
	}
	u[model] = u[model].Add(usage)
	return u
}
