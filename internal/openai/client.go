package openai

import (
	"context"

	"trip2g/internal/features"

	"github.com/sashabaranov/go-openai"
)

// Client wraps OpenAI API for embedding generation.
type Client struct {
	client *openai.Client
	model  features.EmbeddingModel
}

// EmbeddingResult holds the embedding vector and token usage.
type EmbeddingResult struct {
	Vector []float32
	Tokens int
}

// New creates a new OpenAI client.
func New(apiKey string, model features.EmbeddingModel) *Client {
	return &Client{
		client: openai.NewClient(apiKey),
		model:  model,
	}
}

// Model returns the configured embedding model.
func (c *Client) Model() features.EmbeddingModel {
	return c.model
}

// CreateEmbedding generates an embedding for the given text.
func (c *Client) CreateEmbedding(ctx context.Context, text string) (*EmbeddingResult, error) {
	resp, err := c.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Model: openai.EmbeddingModel(c.model.String()),
		Input: []string{text},
	})
	if err != nil {
		return nil, err
	}

	return &EmbeddingResult{
		Vector: resp.Data[0].Embedding,
		Tokens: resp.Usage.TotalTokens,
	}, nil
}
