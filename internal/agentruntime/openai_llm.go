package agentruntime

import (
	"context"

	goopenai "github.com/sashabaranov/go-openai"
)

// OpenAILLM is the production LLM backed by any OpenAI-compatible chat
// completions endpoint. BaseURL is the configurable knob: leave empty for the
// default OpenAI endpoint, or point it at a local/vendor model
// (e.g. "http://localhost:11434/v1" for Ollama).
type OpenAILLM struct {
	client *goopenai.Client
}

// NewOpenAILLM builds an OpenAILLM. apiKey is the bearer credential; baseURL,
// when non-empty, overrides the default endpoint.
func NewOpenAILLM(apiKey, baseURL string) *OpenAILLM {
	cfg := goopenai.DefaultConfig(apiKey)
	if baseURL != "" {
		cfg.BaseURL = baseURL
	}
	return &OpenAILLM{client: goopenai.NewClientWithConfig(cfg)}
}

// Chat issues one chat-completion request with the given tools.
func (l *OpenAILLM) Chat(ctx context.Context, model string, messages []Message, tools []ToolDef) (ChatResult, error) {
	req := goopenai.ChatCompletionRequest{
		Model:    model,
		Messages: toOpenAIMessages(messages),
		Tools:    toOpenAITools(tools),
	}

	resp, err := l.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return ChatResult{}, err
	}

	out := ChatResult{
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
	}
	if len(resp.Choices) == 0 {
		return out, nil
	}
	msg := resp.Choices[0].Message
	out.Content = msg.Content
	for _, tc := range msg.ToolCalls {
		out.ToolCalls = append(out.ToolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}
	return out, nil
}

func toOpenAIMessages(messages []Message) []goopenai.ChatCompletionMessage {
	out := make([]goopenai.ChatCompletionMessage, 0, len(messages))
	for _, m := range messages {
		om := goopenai.ChatCompletionMessage{
			Role:       m.Role,
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
			Name:       m.Name,
		}
		for _, tc := range m.ToolCalls {
			om.ToolCalls = append(om.ToolCalls, goopenai.ToolCall{
				ID:   tc.ID,
				Type: goopenai.ToolTypeFunction,
				Function: goopenai.FunctionCall{
					Name:      tc.Name,
					Arguments: tc.Arguments,
				},
			})
		}
		out = append(out, om)
	}
	return out
}

func toOpenAITools(tools []ToolDef) []goopenai.Tool {
	out := make([]goopenai.Tool, 0, len(tools))
	for _, t := range tools {
		out = append(out, goopenai.Tool{
			Type: goopenai.ToolTypeFunction,
			Function: &goopenai.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		})
	}
	return out
}
