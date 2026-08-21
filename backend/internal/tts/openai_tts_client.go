package tts

import (
	"bytes"
	"context"
	"hkorpo/book/internal/primitive"
	"hkorpo/book/pkg/errorwrapper"
	"io"
	"unicode"

	"github.com/openai/openai-go/v3"
)

// maxTTSInputChars is OpenAI's hard limit on a single audio/speech
// request's input length ("String should have at most 4096 characters").
// A book's narration script routinely exceeds this, so CreateAudioFromString
// splits it into several requests and stitches the resulting WAV files back
// into one (see splitForTTS, wav.go).
const maxTTSInputChars = 4096

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
	var (
		fmtChunk []byte
		data     bytes.Buffer
	)

	for _, chunk := range splitForTTS(content) {
		resp, err := oc.client.Audio.Speech.New(ctx, openai.AudioSpeechNewParams{
			Input:          chunk,
			Model:          oc.model,
			Voice:          openai.AudioSpeechNewParamsVoiceUnion{OfString: openai.String("alloy")},
			ResponseFormat: openai.AudioSpeechNewParamsResponseFormatWAV,
		})
		if err != nil {
			return nil, 0, primitive.ModelUsage{}, errorwrapper.Wrap(err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, 0, primitive.ModelUsage{}, errorwrapper.Wrap(err)
		}

		wav, err := parseWAV(body)
		if err != nil {
			return nil, 0, primitive.ModelUsage{}, errorwrapper.Wrap(err)
		}

		if fmtChunk == nil {
			fmtChunk = wav.fmtChunk
		}
		data.Write(wav.data)
	}

	audio := buildWAV(fmtChunk, data.Bytes())

	// The classic audio/speech endpoint doesn't return usage counts, and
	// oc.model is priced per input character rather than per token (see
	// cmd/generate_tts/main.go), so InputTokens approximates the character
	// count of the (whole, unsplit) input text instead of an actual token
	// count.
	usage := primitive.ModelUsage{InputTokens: int64(len([]rune(content)))}

	return io.NopCloser(bytes.NewReader(audio)), int64(len(audio)), usage, nil
}

// splitForTTS splits text into chunks of at most maxTTSInputChars runes,
// breaking on whitespace where possible so a request never cuts a word in
// half. A single "word" longer than the limit is hard-split as a last
// resort, which never happens in practice for narration prose.
func splitForTTS(text string) []string {
	runes := []rune(text)
	if len(runes) == 0 {
		return []string{text}
	}

	var chunks []string
	for len(runes) > 0 {
		if len(runes) <= maxTTSInputChars {
			chunks = append(chunks, string(runes))
			break
		}

		splitAt := maxTTSInputChars
		for splitAt > 0 && !unicode.IsSpace(runes[splitAt]) {
			splitAt--
		}
		if splitAt == 0 {
			splitAt = maxTTSInputChars
		}

		chunks = append(chunks, string(runes[:splitAt]))
		runes = runes[splitAt:]
		if len(runes) > 0 && unicode.IsSpace(runes[0]) {
			runes = runes[1:]
		}
	}
	return chunks
}
