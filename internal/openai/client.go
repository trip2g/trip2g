package openai

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

// Client wraps an OpenAI-compatible API for embedding generation.
type Client struct {
	client    *openai.Client
	modelName string
}

// EmbeddingResult holds the embedding vector and token usage.
type EmbeddingResult struct {
	Vector []float32
	Tokens int
}

// New creates a new OpenAI-compatible embedding client.
// modelName is the model identifier sent in API requests (e.g. "text-embedding-3-small"
// or any name accepted by the server).
// baseURL is optional; when non-empty it overrides the default OpenAI endpoint
// (e.g. "http://localhost:8080/v1" for a local TEI server).
func New(apiKey string, modelName string, baseURL string) *Client {
	cfg := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		cfg.BaseURL = baseURL
	}
	return &Client{
		client:    openai.NewClientWithConfig(cfg),
		modelName: modelName,
	}
}

// ModelName returns the configured embedding model name.
func (c *Client) ModelName() string {
	return c.modelName
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
		Model: openai.EmbeddingModel(c.modelName),
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
