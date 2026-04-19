package main

import (
	"context"
	"fmt"
	"io"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type OpenAIProvider struct {
	client *openai.Client
	model  string
}

func newOpenAIProvider(apiKey, model string) *OpenAIProvider {
	opts := []option.RequestOption{}
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	client := openai.NewClient(opts...)
	return &OpenAIProvider{client: &client, model: model}
}

func (p *OpenAIProvider) Stream(ctx context.Context, system, prompt string, w io.Writer) error {
	messages := []openai.ChatCompletionMessageParamUnion{}
	if system != "" {
		messages = append(messages, openai.SystemMessage(system))
	}
	messages = append(messages, openai.UserMessage(prompt))

	stream := p.client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(p.model),
		Messages: messages,
	})
	for stream.Next() {
		chunk := stream.Current()
		if len(chunk.Choices) > 0 {
			fmt.Fprint(w, chunk.Choices[0].Delta.Content)
		}
	}
	if err := stream.Err(); err != nil {
		return fmt.Errorf("openai stream: %w", err)
	}
	return nil
}
