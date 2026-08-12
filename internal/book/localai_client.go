package book

import (
	"context"
	"hkorpo/book/internal/platform/localai"
	"io"
)

type LocalAiClient struct {
	client *localai.Client
}

func NewLocalAiClient(client *localai.Client) *LocalAiClient {
	return &LocalAiClient{
		client: client,
	}
}

func (oc *LocalAiClient) CreateAudioFromString(ctx context.Context, content string) (io.ReadCloser, int64, error) {
	resp, err := oc.client.Request(ctx, localai.TTS, map[string]string{
		"input": content,
		"model": "kokoro",
	})

	if err != nil {
		return nil, 0, err
	}

	return resp.Body, resp.ContentLength, nil
}
