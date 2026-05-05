package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"trip2g/internal/model"

	"golang.org/x/sync/errgroup"
)

type federationFanoutEnv interface {
	model.FederationClientFactory
}

type ProxiedResult struct {
	KB      *model.MCPFederationNote
	Result  model.FederationResult
	Err     error
	Latency time.Duration
}

func fanout(
	ctx context.Context,
	env federationFanoutEnv,
	kbs []*model.MCPFederationNote,
	call func(ctx context.Context, client model.Federation) (model.FederationResult, error),
) []ProxiedResult {
	results := make([]ProxiedResult, len(kbs))
	var g errgroup.Group

	for i, kb := range kbs {
		results[i].KB = kb
		g.Go(func() error {
			start := time.Now()
			client, err := env.FederationClient(ctx, kb.ID)
			if err == nil {
				results[i].Result, err = call(ctx, client)
			}
			results[i].Err = err
			results[i].Latency = time.Since(start)
			return nil
		})
	}

	_ = g.Wait()
	return results
}

func selectFederationKBs(kbs []*model.MCPFederationNote, kbID string, kbIDs []string) []*model.MCPFederationNote {
	if kbID == "" && len(kbIDs) == 0 {
		return kbs
	}

	allowed := make(map[string]struct{}, 1+len(kbIDs))
	if kbID != "" {
		head, _ := splitKBID(kbID)
		allowed[head] = struct{}{}
	}
	for _, id := range kbIDs {
		head, _ := splitKBID(id)
		allowed[head] = struct{}{}
	}

	result := make([]*model.MCPFederationNote, 0, len(kbs))
	for _, kb := range kbs {
		if _, ok := allowed[kb.ID]; ok {
			result = append(result, kb)
		}
	}
	return result
}

func findFederationKB(kbs []*model.MCPFederationNote, kbID string) (*model.MCPFederationNote, string) {
	head, rest := splitKBID(kbID)
	for _, kb := range kbs {
		if kb.ID == head {
			return kb, rest
		}
	}
	return nil, rest
}

func federationResultToToolResult(result model.FederationResult) CallToolResult {
	content := make([]Content, 0, len(result.Content))
	for _, item := range result.Content {
		content = append(content, Content{Type: item.Type, Text: item.Text})
	}

	var structured any
	if len(result.StructuredContent) > 0 {
		structured = result.StructuredContent
	}
	return CallToolResult{
		Content:           content,
		StructuredContent: structured,
		IsError:           result.IsError,
	}
}

func aggregateFederationResults(results []ProxiedResult) CallToolResult {
	payload := FederatedCallPayload{}
	var text strings.Builder
	for _, result := range results {
		kbID := ""
		if result.KB != nil {
			kbID = result.KB.ID
		}
		if result.Err != nil {
			payload.Errors = append(payload.Errors, FederatedCallError{
				KBID:  kbID,
				Error: result.Err.Error(),
			})
			continue
		}
		payload.Results = append(payload.Results, FederatedCallResult{
			KBID:    kbID,
			Result:  result.Result,
			Latency: result.Latency.String(),
		})
		for _, content := range result.Result.Content {
			if content.Text == "" {
				continue
			}
			if text.Len() > 0 {
				text.WriteString("\n\n")
			}
			if kbID != "" {
				_, _ = fmt.Fprintf(&text, "[%s]\n", kbID)
			}
			text.WriteString(content.Text)
		}
	}
	if text.Len() == 0 {
		text.WriteString("No federation results")
	}

	toolResult := structuredToolResult(text.String(), payload)
	toolResult.IsError = len(payload.Results) == 0 && len(payload.Errors) > 0
	return toolResult
}
