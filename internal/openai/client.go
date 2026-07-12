package openai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"

	"trip2g/internal/metrics"
)

// Client wraps an OpenAI-compatible API for embedding generation.
type Client struct {
	client    *openai.Client
	modelName string
	metrics   *metrics.EmbeddingMetrics
}

// EmbeddingResult holds the embedding vector and token usage.
type EmbeddingResult struct {
	Vector []float32
	Tokens int
}

// Bounded kind labels for embedding requests — what the text being embedded is.
const (
	KindWholeNote = "whole_note"
	KindChunk     = "chunk"
	KindQuery     = "query"
	KindDebug     = "debug"
)

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

// SetMetrics wires the embedding request/duration metrics. Optional — a
// client without metrics set (e.g. in tests) records nothing.
func (c *Client) SetMetrics(m *metrics.EmbeddingMetrics) {
	c.metrics = m
}

// ModelName returns the configured embedding model name.
func (c *Client) ModelName() string {
	return c.modelName
}

// CreateEmbedding generates an embedding for the given text. kind labels the
// request in metrics (see the Kind* constants).
func (c *Client) CreateEmbedding(ctx context.Context, text string, kind string) (*EmbeddingResult, error) {
	results, err := c.CreateEmbeddings(ctx, []string{text}, kind)
	if err != nil {
		return nil, err
	}
	return &results[0], nil
}

// CreateEmbeddings generates embeddings for multiple texts in a single API call.
// kind labels the request in metrics (see the Kind* constants).
func (c *Client) CreateEmbeddings(ctx context.Context, texts []string, kind string) ([]EmbeddingResult, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	start := time.Now()
	resp, err := c.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Model: openai.EmbeddingModel(c.modelName),
		Input: texts,
	})
	seconds := time.Since(start).Seconds()
	if err != nil {
		c.metrics.RecordEmbeddingRequest(kind, "error", classifyError(err), seconds)
		return nil, err
	}
	if len(resp.Data) != len(texts) {
		c.metrics.RecordEmbeddingRequest(kind, "error", metrics.EmbeddingErrorOther, seconds)
		return nil, fmt.Errorf("expected %d embeddings, got %d", len(texts), len(resp.Data))
	}
	c.metrics.RecordEmbeddingRequest(kind, "ok", "", seconds)

	results := make([]EmbeddingResult, len(texts))
	for i, d := range resp.Data {
		results[i] = EmbeddingResult{
			Vector: d.Embedding,
			Tokens: resp.Usage.TotalTokens / len(texts), // approximate: API returns only total, not per-input
		}
	}
	return results, nil
}

// classifyError maps an embedding API error to a bounded reason label. The
// batch-too-large and rate-limit (overloaded) cases are the ones that caused
// the queue-saturation incident this metric exists to catch early.
func classifyError(err error) string {
	var apiErr *openai.APIError
	if errors.As(err, &apiErr) {
		if apiErr.HTTPStatusCode == http.StatusTooManyRequests {
			return metrics.EmbeddingErrorOverloaded
		}
		if apiErr.HTTPStatusCode == http.StatusBadRequest && looksLikeBatchTooLarge(apiErr.Message) {
			return metrics.EmbeddingErrorBatchTooLarge
		}
		return metrics.EmbeddingErrorOther
	}
	var reqErr *openai.RequestError
	if errors.As(err, &reqErr) {
		if reqErr.HTTPStatusCode == http.StatusTooManyRequests {
			return metrics.EmbeddingErrorOverloaded
		}
	}
	return metrics.EmbeddingErrorOther
}

func looksLikeBatchTooLarge(message string) bool {
	msg := strings.ToLower(message)
	return strings.Contains(msg, "batch") || strings.Contains(msg, "too many") || strings.Contains(msg, "too large")
}
