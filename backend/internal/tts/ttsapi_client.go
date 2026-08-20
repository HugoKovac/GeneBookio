package tts

import (
	"context"
	"hkorpo/book/internal/platform/ttsapi"
	"hkorpo/book/internal/primitive"
	"io"
)

type TTSApiClient struct {
	client *ttsapi.Client
}

func NewTTSApiClient(client *ttsapi.Client) *TTSApiClient {
	return &TTSApiClient{
		client: client,
	}
}

func (oc *TTSApiClient) CreateAudioFromString(ctx context.Context, content string, language primitive.Language) (io.ReadCloser, int64, error) {
	resp, err := oc.client.Request(ctx, ttsapi.TTS, map[string]string{
		"input":    content,
		"language": language.String(),
	})

	if err != nil {
		return nil, 0, err
	}

	return resp.Body, resp.ContentLength, nil
}
