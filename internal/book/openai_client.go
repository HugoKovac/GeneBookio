package book

import (
	"context"
	"fmt"
	"hkorpo/book/pkg/errorwrapper"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

type OpenAiClient struct {
	client openai.Client
}

func NewOpenAiClient(client openai.Client) *OpenAiClient {
	return &OpenAiClient{
		client: client,
	}
}

func (oc *OpenAiClient) Request(ctx context.Context, request string) (string, error) {
	preparation, err := oc.client.Responses.New(ctx, responses.ResponseNewParams{
		Model: openai.ChatModelGPT5_2,
		Input: responses.ResponseNewParamsInputUnion{OfString: openai.String(
			request,
		)},
	})
	if err != nil {
		return "", errorwrapper.Wrap(fmt.Errorf("generate preparation: %v", err))
	}
	return preparation.OutputText(), nil
}
