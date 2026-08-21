package tts

import (
	"context"
	"hkorpo/book/internal/primitive"
	"hkorpo/book/pkg/errorwrapper"
	"io"

	"github.com/openai/openai-go/v3"
)

// OpenAiTTSClient synthesizes audio via OpenAI's audio/speech endpoint.
type OpenAiTTSClient struct {
	client openai.Client
	model  openai.SpeechModel
}

func NewOpenAiTTSClient(client openai.Client, model openai.SpeechModel) *OpenAiTTSClient {
	return &OpenAiTTSClient{client: client, model: model}
}

func (oc *OpenAiTTSClient) ModelName() string {
	return oc.model
}

func (oc *OpenAiTTSClient) CreateAudioFromString(ctx context.Context, content string, _ primitive.Language) (io.ReadCloser, int64, primitive.ModelUsage, error) {
	resp, err := oc.client.Audio.Speech.New(ctx, openai.AudioSpeechNewParams{
		Input:          content,
		Model:          oc.model,
		Voice:          openai.AudioSpeechNewParamsVoiceUnion{OfString: openai.String("alloy")},
		ResponseFormat: openai.AudioSpeechNewParamsResponseFormatWAV,
	})
	if err != nil {
		return nil, 0, primitive.ModelUsage{}, errorwrapper.Wrap(err)
	}

	// The classic audio/speech endpoint doesn't return usage counts, and
	// oc.model is priced per input character rather than per token (see
	// cmd/generate_tts/main.go), so InputTokens approximates the character
	// count of the input text instead of an actual token count.
	usage := primitive.ModelUsage{InputTokens: int64(len([]rune(content)))}

	return resp.Body, resp.ContentLength, usage, nil
}
