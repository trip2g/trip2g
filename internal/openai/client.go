package openai

import (
	"context"
	"fmt"

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

// New creates a new OpenAI-compatible client.
// baseURL is optional; when non-empty it overrides the default OpenAI endpoint
// (e.g. "http://localhost:11434/v1" for a local Ollama instance).
func New(apiKey string, model features.EmbeddingModel, baseURL string) *Client {
	cfg := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		cfg.BaseURL = baseURL
	}
	return &Client{
		client: openai.NewClientWithConfig(cfg),
		model:  model,
	}
}

// Model returns the configured embedding model.
func (c *Client) Model() features.EmbeddingModel {
	return c.model
}

// CreateEmbedding generates an embedding for the given text.
func (c *Client) CreateEmbedding(ctx context.Context, text string) (*EmbeddingResult, error) {
	results, err := c.CreateEmbeddings(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return &results[0], nil
}

// CreateEmbeddings generates embeddings for multiple texts in a single API call.
func (c *Client) CreateEmbeddings(ctx context.Context, texts []string) ([]EmbeddingResult, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	resp, err := c.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Model: openai.EmbeddingModel(c.model.String()),
		Input: texts,
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Data) != len(texts) {
		return nil, fmt.Errorf("expected %d embeddings, got %d", len(texts), len(resp.Data))
	}
	results := make([]EmbeddingResult, len(texts))
	for i, d := range resp.Data {
		results[i] = EmbeddingResult{
			Vector: d.Embedding,
			Tokens: resp.Usage.TotalTokens / len(texts), // approximate: API returns only total, not per-input
		}
	}
	return results, nil
}
