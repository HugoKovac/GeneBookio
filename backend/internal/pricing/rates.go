package pricing

import "hkorpo/book/internal/primitive"

// ModelRate is a model's USD price per 1,000,000 input/output units, taken
// from https://platform.openai.com/docs/pricing. For tts-1/tts-1-hd, which
// OpenAI bills per input character rather than per token, InputPerMillion is
// USD per 1M input characters and OutputPerMillion is unused (0) — see
// internal/tts.OpenAiTTSClient, which stores the character count where the
// other clients store a token count.
type ModelRate struct {
	InputPerMillion  float64
	OutputPerMillion float64
}

// usdRatesPerMillion covers the models cmd/prepare_chapters, cmd/generate_script
// and cmd/generate_tts hardcode. A model missing from this table (e.g. the
// "test-mode" substitution clients, or a model swapped in later without
// updating this table) simply prices at $0 rather than erroring.
var usdRatesPerMillion = map[string]ModelRate{
	"gpt-5":      {InputPerMillion: 1.25, OutputPerMillion: 10.00},
	"gpt-5-mini": {InputPerMillion: 0.25, OutputPerMillion: 2.00},
	"gpt-5.2":    {InputPerMillion: 1.75, OutputPerMillion: 14.00},
	"tts-1":      {InputPerMillion: 15.00},
	"tts-1-hd":   {InputPerMillion: 30.00},
}

// CostUSD sums the USD cost of every model in usage using usdRatesPerMillion.
func CostUSD(usage primitive.TokenUsage) float64 {
	var total float64
	for model, u := range usage {
		rate, ok := usdRatesPerMillion[model]
		if !ok {
			continue
		}
		total += float64(u.InputTokens)/1_000_000*rate.InputPerMillion +
			float64(u.OutputTokens)/1_000_000*rate.OutputPerMillion
	}
	return total
}
