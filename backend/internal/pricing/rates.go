package pricing

import "hkorpo/book/internal/primitive"

// ModelRate is a model's USD price per 1,000,000 input/output units, taken
// from https://platform.openai.com/docs/pricing. For tts-1/tts-1-hd, which
// OpenAI bills per input character rather than per token, InputPerMillion is
// USD per 1M input characters and OutputPerMillion is unused (0) — see
// internal/tts.OpenAiTTSClient, which stores the character count where the
// other clients store a token count. gpt-4o-mini-tts, by contrast, is priced
// like a regular text model: InputPerMillion is USD per 1M input text
// tokens and OutputPerMillion is USD per 1M output audio tokens (the
// dominant cost, roughly proportional to spoken duration) — see
// EstimateTTSInputUsage, which picks the right unit for a given model.
type ModelRate struct {
	InputPerMillion  float64
	OutputPerMillion float64
}

// usdRatesPerMillion covers the models cmd/prepare_chapters, cmd/generate_script
// and cmd/generate_tts hardcode. A model missing from this table (e.g. the
// "test-mode" substitution clients, or a model swapped in later without
// updating this table) simply prices at $0 rather than erroring.
var usdRatesPerMillion = map[string]ModelRate{
	"gpt-5":           {InputPerMillion: 1.25, OutputPerMillion: 10.00},
	"gpt-5-mini":      {InputPerMillion: 0.25, OutputPerMillion: 2.00},
	"gpt-5.2":         {InputPerMillion: 1.75, OutputPerMillion: 14.00},
	"tts-1":           {InputPerMillion: 15.00},
	"tts-1-hd":        {InputPerMillion: 30.00},
	"gpt-4o-mini-tts": {InputPerMillion: 0.60, OutputPerMillion: 12.00},
}

// EstimateTTSInputUsage builds the pre-call usage estimate CheckBudget needs
// to guard a TTS request's input cost before it's sent (see
// internal/tts.Service.CreateAudioFromScript). Character-priced models
// (tts-1, tts-1-hd — no OutputPerMillion price in usdRatesPerMillion) bill
// per input character, so InputTokens holds an exact character count.
// Token-priced models (e.g. gpt-4o-mini-tts) bill input like any other text
// model, so InputTokens holds the same ~4-chars-per-token estimate
// EstimateTokens uses elsewhere. Either way this only estimates the input
// side: a token-priced model's dominant cost is its output (audio) tokens,
// which aren't known until the API reports real usage — CheckBudget's
// post-call pass is what catches an output-driven overrun.
func EstimateTTSInputUsage(model, text string) primitive.ModelUsage {
	if rate, ok := usdRatesPerMillion[model]; ok && rate.OutputPerMillion > 0 {
		return primitive.ModelUsage{InputTokens: EstimateTokens(text)}
	}
	return primitive.ModelUsage{InputTokens: int64(len([]rune(text)))}
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
