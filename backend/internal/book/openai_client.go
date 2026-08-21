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

func (sc *SubstitutionAiClient) Request(_ context.Context, request string, _ int64) (string, primitive.ModelUsage, error) {
	return fmt.Sprintf("==START OF REQUEST==\n%s\n==END OF REQUEST==", request), primitive.ModelUsage{}, nil
}

func (oc *OpenAiClient) ModelName() string {
	return string(oc.model)
}

// Request calls the Responses API. maxOutputTokens, when positive, caps the
// response length server-side — see pricing.Calculator.CapOutputTokens,
// which sizes it from the caller's remaining EUR budget so a single request
// can't blow past it.
func (oc *OpenAiClient) Request(ctx context.Context, request string, maxOutputTokens int64) (string, primitive.ModelUsage, error) {
	params := responses.ResponseNewParams{
		Model: oc.model,
		Input: responses.ResponseNewParamsInputUnion{OfString: openai.String(
			request,
		)},
	}
	if maxOutputTokens > 0 {
		params.MaxOutputTokens = openai.Int(maxOutputTokens)
	}

	preparation, err := oc.client.Responses.New(ctx, params)
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
