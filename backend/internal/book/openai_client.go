package book

import (
	"context"
	"fmt"
	"hkorpo/book/internal/primitive"
	"hkorpo/book/pkg/errorwrapper"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

type ConfigAi struct {
	TEST_MODE bool `envconfig:"AI_TEST_MODE"`
}

type OpenAiClient struct {
	client openai.Client
	model  openai.ChatModel
}

func NewOpenAiClient(client openai.Client, model openai.ChatModel) *OpenAiClient {
	return &OpenAiClient{
		client: client,
		model:  model,
	}
}

// SubstitutionAiClient stands in for OpenAiClient when ConfigAi.TEST_MODE is
// enabled, so pipeline stages can be exercised without spending real AI tokens.
type SubstitutionAiClient struct{}

func NewSubstitutionAiClient() *SubstitutionAiClient {
	return &SubstitutionAiClient{}
}

func (sc *SubstitutionAiClient) ModelName() string {
	return "test-mode"
}

func (sc *SubstitutionAiClient) Request(_ context.Context, request string) (string, primitive.ModelUsage, error) {
	return fmt.Sprintf("==START OF REQUEST==\n%s\n==END OF REQUEST==", request), primitive.ModelUsage{}, nil
}

func (oc *OpenAiClient) ModelName() string {
	return string(oc.model)
}

func (oc *OpenAiClient) Request(ctx context.Context, request string) (string, primitive.ModelUsage, error) {
	preparation, err := oc.client.Responses.New(ctx, responses.ResponseNewParams{
		Model: oc.model,
		Input: responses.ResponseNewParamsInputUnion{OfString: openai.String(
			request,
		)},
	})
	if err != nil {
		return "", primitive.ModelUsage{}, errorwrapper.Wrap(fmt.Errorf("generate preparation: %v", err))
	}
	usage := primitive.ModelUsage{
		InputTokens:  preparation.Usage.InputTokens,
		OutputTokens: preparation.Usage.OutputTokens,
		TotalTokens:  preparation.Usage.TotalTokens,
	}
	return preparation.OutputText(), usage, nil
}
