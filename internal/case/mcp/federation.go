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
	concurrency int,
	timeout time.Duration,
	call func(ctx context.Context, client model.Federation) (model.FederationResult, error),
) []ProxiedResult {
	results := make([]ProxiedResult, len(kbs))
	var g errgroup.Group
	if concurrency > 0 {
		g.SetLimit(concurrency)
	}

	for i, kb := range kbs {
		results[i].KB = kb
		g.Go(func() error {
			start := time.Now()
			results[i].Result, results[i].Err = callPeer(ctx, env, kb.ID, timeout, call)
			results[i].Latency = time.Since(start)
			return nil
		})
	}

	_ = g.Wait()
	return results
}

// callPeer calls one peer under a per-peer timeout. The call runs in its own
// goroutine so a hung client that ignores ctx cancellation still releases the
// fan-out concurrency slot when the timeout fires.
func callPeer(
	ctx context.Context,
	env federationFanoutEnv,
	kbID string,
	timeout time.Duration,
	call func(ctx context.Context, client model.Federation) (model.FederationResult, error),
) (model.FederationResult, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	type outcome struct {
		result model.FederationResult
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		client, err := env.FederationClient(ctx, kbID)
		var result model.FederationResult
		if err == nil {
			result, err = call(ctx, client)
		}
		done <- outcome{result: result, err: err}
	}()

	select {
	case o := <-done:
		return o.result, o.err
	case <-ctx.Done():
		return model.FederationResult{}, ctx.Err()
	}
}

// capBlindFanout caps a blind fan-out at limit peers (registration order),
// returning the peers to query and those skipped for reporting.
func capBlindFanout(kbs []*model.MCPFederationNote, limit int) ([]*model.MCPFederationNote, []*model.MCPFederationNote) {
	if limit > 0 && len(kbs) > limit {
		return kbs[:limit], kbs[limit:]
	}
	return kbs, nil
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

func aggregateFederationResults(results []ProxiedResult, skipped []*model.MCPFederationNote) CallToolResult {
	payload := FederatedCallPayload{}
	for _, kb := range skipped {
		payload.Skipped = append(payload.Skipped, FederatedCallSkipped{
			KBID:   kb.ID,
			Reason: "fanout_limit",
		})
	}
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
	if len(skipped) > 0 {
		ids := make([]string, 0, len(skipped))
		for _, kb := range skipped {
			ids = append(ids, kb.ID)
		}
		_, _ = fmt.Fprintf(&text, "\n\n%d bases were not queried due to the fan-out limit — query them directly with kb_id: %s",
			len(ids), strings.Join(ids, ", "))
	}

	toolResult := structuredToolResult(text.String(), payload)
	toolResult.IsError = len(payload.Results) == 0 && len(payload.Errors) > 0
	return toolResult
}
