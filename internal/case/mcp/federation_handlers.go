package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"trip2g/internal/model"
)

func handleFederatedSearch(ctx context.Context, env Env, id any, argsRaw json.RawMessage) Response {
	args, errResp := unmarshalArgs[model.MCPSearchParams](argsRaw, id, "federated_search")
	if errResp != nil {
		return *errResp
	}
	if args.Query == "" {
		return errorResponse(id, ErrCodeInvalidParams, "query is required")
	}
	if depthErr := federationPathDepthExceeded(ctx, env, id, args.KBID); depthErr != nil {
		return *depthErr
	}

	kbs, err := accessibleKBNotes(ctx, env)
	if err != nil {
		return errorResponse(id, ErrCodeInternal, err.Error())
	}
	if len(kbs) == 0 {
		return federationNotConfiguredResponse(id, args.KBID, kbs)
	}

	if args.KBID == "" {
		selected := selectFederationKBs(kbs, "", args.KBIDs)
		if len(selected) == 0 {
			return federationNotConfiguredResponse(id, "", kbs)
		}
		// The limit only caps blind fan-outs; explicit kb_ids targeting is precise.
		var skipped []*model.MCPFederationNote
		if len(args.KBIDs) == 0 {
			selected, skipped = capBlindFanout(selected, env.FederatedFanoutLimit())
		}
		results := fanout(ctx, env, selected, env.FederatedFanoutConcurrency(), env.FederatedFanoutTimeout(),
			func(ctx context.Context, client model.Federation) (model.FederationResult, error) {
				return client.Search(ctx, forwarded(*args))
			})
		m := metricsFromContext(ctx)
		touched := 0
		for _, r := range results {
			m.RecordFederatedRequest(federatedStatus(r.Err))
			if r.Err == nil {
				touched++
			}
		}
		m.ObserveFanoutBases(touched)
		return successResponse(id, aggregateFederationResults(results, skipped, kbIDs(kbs)))
	}

	kb, rest := findFederationKB(kbs, args.KBID)
	if kb == nil {
		return federationNotConfiguredResponse(id, args.KBID, kbs)
	}
	var client model.Federation
	client, err = env.FederationClient(ctx, kb.ID)
	if err != nil {
		// A client that can't be built is still a failed outbound attempt,
		// same as on the fan-out path.
		metricsFromContext(ctx).RecordFederatedRequest(federatedStatus(err))
		return errorResponse(id, ErrCodeInternal, err.Error())
	}
	params := forwarded(*args)
	var result model.FederationResult
	if rest == "" {
		result, err = client.Search(ctx, params)
	} else {
		params.KBID = rest
		result, err = client.FederatedSearch(ctx, params)
	}
	metricsFromContext(ctx).RecordFederatedRequest(federatedStatus(err))
	if err != nil {
		return errorResponse(id, ErrCodeInternal, err.Error())
	}
	rewriteFederatedResponse(kb.ID, kbIDs(kbs), &result)
	return successResponse(id, federationResultToToolResult(result))
}

// callFederatedSingleKB handles the common pattern: look up KB → get client → call → handle error → return.
// The call closure receives the client and the "rest" portion of the KB ID (empty if direct, non-empty if federated).
func callFederatedSingleKB(
	ctx context.Context,
	env Env,
	id any,
	kbID string,
	call func(client model.Federation, rest string) (model.FederationResult, error),
) Response {
	result, errResp := federatedSingleKBResult(ctx, env, id, kbID, call)
	if errResp != nil {
		return *errResp
	}
	return successResponse(id, federationResultToToolResult(result))
}

// federatedSingleKBResult is the shared core of callFederatedSingleKB that hands
// back the raw result so callers can post-process it (e.g. cache it) before
// building the tool response. On any failure it returns the error Response.
func federatedSingleKBResult(
	ctx context.Context,
	env Env,
	id any,
	kbID string,
	call func(client model.Federation, rest string) (model.FederationResult, error),
) (model.FederationResult, *Response) {
	if errResp := federationPathDepthExceeded(ctx, env, id, kbID); errResp != nil {
		return model.FederationResult{}, errResp
	}
	kbs, err := accessibleKBNotes(ctx, env)
	if err != nil {
		resp := errorResponse(id, ErrCodeInternal, err.Error())
		return model.FederationResult{}, &resp
	}
	if len(kbs) == 0 {
		resp := federationNotConfiguredResponse(id, kbID, kbs)
		return model.FederationResult{}, &resp
	}
	kb, rest := findFederationKB(kbs, kbID)
	if kb == nil {
		resp := federationNotConfiguredResponse(id, kbID, kbs)
		return model.FederationResult{}, &resp
	}
	// Bound the hop like the fan-out path does. Without a deadline a peer that
	// accepts the connection and never answers leaves the agent waiting on a
	// tool that never returns, which is worse for it than a fast failure.
	result, err := callPeer(ctx, env, kb.ID, env.FederatedFanoutTimeout(),
		func(ctx context.Context, client model.Federation) (model.FederationResult, error) {
			return call(client, rest)
		})
	metricsFromContext(ctx).RecordFederatedRequest(federatedStatus(err))
	if err != nil {
		resp := errorResponse(id, ErrCodeInternal, err.Error())
		return model.FederationResult{}, &resp
	}
	// A peer answering "not configured" names the kb_id in its own frame; the
	// caller must see it in theirs, same as on the fan-out path.
	rewriteFederatedResponse(kb.ID, kbIDs(kbs), &result)
	return result, nil
}

// handleFederatedInstructions fetches a federated base's own instructions by
// kb_id, forwarding through the federation hop chain like the other federated
// tools. Results are cached per full kb_id path so repeat calls (and
// already-visited routes) are served without re-forwarding.
func handleFederatedInstructions(ctx context.Context, env Env, id any, argsRaw json.RawMessage) Response {
	args, errResp := unmarshalArgs[model.MCPInstructionsParams](argsRaw, id, "federated_instructions")
	if errResp != nil {
		return *errResp
	}
	if args.KBID == "" {
		return errorResponse(id, ErrCodeInvalidParams, "kb_id is required")
	}

	if cached, ok := env.CachedFederatedInstructions(args.KBID); ok {
		return successResponse(id, federationResultToToolResult(cached))
	}

	result, errResp := federatedSingleKBResult(ctx, env, id, args.KBID,
		func(client model.Federation, rest string) (model.FederationResult, error) {
			if rest == "" {
				return client.Instructions(ctx)
			}
			return client.FederatedInstructions(ctx, model.MCPInstructionsParams{KBID: rest})
		})
	if errResp != nil {
		return *errResp
	}
	env.StoreFederatedInstructions(args.KBID, result)
	return successResponse(id, federationResultToToolResult(result))
}

func handleFederatedSimilar(ctx context.Context, env Env, id any, argsRaw json.RawMessage) Response {
	args, errResp := unmarshalArgs[model.MCPSimilarParams](argsRaw, id, "federated_similar")
	if errResp != nil {
		return *errResp
	}
	if args.KBID == "" {
		return errorResponse(id, ErrCodeInvalidParams, "kb_id is required")
	}
	params := forwarded(*args)
	return callFederatedSingleKB(ctx, env, id, args.KBID, func(client model.Federation, rest string) (model.FederationResult, error) {
		if rest == "" {
			return client.Similar(ctx, params)
		}
		params.KBID = rest
		return client.FederatedSimilar(ctx, params)
	})
}

func handleFederatedNoteHTML(ctx context.Context, env Env, id any, argsRaw json.RawMessage) Response {
	args, errResp := unmarshalArgs[model.MCPNoteHTMLParams](argsRaw, id, "federated_note_html")
	if errResp != nil {
		return *errResp
	}
	if args.KBID == "" {
		return errorResponse(id, ErrCodeInvalidParams, "kb_id is required")
	}
	params := forwarded(*args)
	return callFederatedSingleKB(ctx, env, id, args.KBID, func(client model.Federation, rest string) (model.FederationResult, error) {
		if rest == "" {
			return client.NoteHTML(ctx, params)
		}
		params.KBID = rest
		return client.FederatedNoteHTML(ctx, params)
	})
}

func handleFederatedExpand(ctx context.Context, env Env, id any, argsRaw json.RawMessage) Response {
	args, errResp := unmarshalArgs[model.MCPExpandParams](argsRaw, id, "federated_expand")
	if errResp != nil {
		return *errResp
	}
	if args.KBID == "" {
		return errorResponse(id, ErrCodeInvalidParams, "kb_id is required")
	}
	params := forwarded(*args)
	return callFederatedSingleKB(ctx, env, id, args.KBID, func(client model.Federation, rest string) (model.FederationResult, error) {
		if rest == "" {
			return client.Expand(ctx, params)
		}
		params.KBID = rest
		return client.FederatedExpand(ctx, params)
	})
}

func handleFederatedGraphQLRequest(ctx context.Context, env Env, id any, argsRaw json.RawMessage) Response {
	if !env.FederatedGraphQLEnabled() {
		return errorResponse(id, ErrCodeMethodNotFound, "Method not found: federated_graphql_request")
	}

	args, errResp := unmarshalArgs[FederatedGraphQLRequestArguments](argsRaw, id, "federated_graphql_request")
	if errResp != nil {
		return *errResp
	}
	if args.KBID == "" {
		return errorResponse(id, ErrCodeInvalidParams, "kb_id is required")
	}
	if args.Query == "" {
		return errorResponse(id, ErrCodeInvalidParams, "query is required")
	}

	if err := validateReadOnlyQuery(args.Query, graphqlFederatedRootFields); err != nil {
		return errorResponse(id, ErrCodeInvalidParams, "query rejected: "+err.Error())
	}

	return callFederatedSingleKB(ctx, env, id, args.KBID, func(client model.Federation, rest string) (model.FederationResult, error) {
		return client.GraphQLRequest(ctx, model.MCPGraphQLParams{KBID: rest, Query: args.Query, Variables: args.Variables})
	})
}

// federationPathDepthExceeded rejects an explicit nested kb_id up front when the
// path cannot fit under the federation depth limit, before any outbound hop fires.
// The bound is invariant along the path: each hop consumes one segment and gains
// one depth unit, so incomingDepth + segmentCount(kbID) is constant from the entry
// node to the leaf. The entry node sees the full path and rejects here, so no
// doomed hop runs and the caller gets a single clean error instead of a
// triple-wrapped -32603. Fan-out and single-segment (segmentCount < 2) calls defer
// to the runtime header-depth backstop in the endpoint: a single segment can only
// exceed when max < 1, and any deeper path is already caught at the entry node.
func federationPathDepthExceeded(ctx context.Context, env Env, id any, kbID string) *Response {
	segs := segmentCount(kbID)
	if segs < 2 {
		return nil
	}
	total := FederationDepthFromContext(ctx) + segs
	if limit := env.FederationMaxDepth(); total > limit {
		resp := errorResponse(id, ErrCodeInternal,
			fmt.Sprintf("kb_id path depth %d exceeds federation max depth %d", total, limit))
		return &resp
	}
	return nil
}

// segmentCount counts the '/'-separated non-empty segments of a kb_id path
// (empty kb_id = 0).
func segmentCount(kbID string) int {
	n := 0
	for _, seg := range strings.Split(kbID, "/") {
		if seg != "" {
			n++
		}
	}
	return n
}

// federationNotConfiguredResponse answers a kb_id this hub cannot route. The
// bases it can route to travel with it, so the caller (or the hop rewriting
// this on the way back) can say what exists instead of what was sent.
func federationNotConfiguredResponse(id any, kbID string, kbs []*model.MCPFederationNote) Response {
	payload := FederationStatusPayload{
		Status:         "federation_not_configured",
		KBID:           kbID,
		ConnectedKBIDs: kbIDs(kbs),
	}
	payload.Message = notConfiguredMessage(payload)
	return successResponse(id, structuredToolResult(payload.Message, payload))
}

// kbIDs lists the bases as the caller may address them; kbs has already been
// filtered by what the caller may read.
func kbIDs(kbs []*model.MCPFederationNote) []string {
	if len(kbs) == 0 {
		return nil
	}
	ids := make([]string, 0, len(kbs))
	for _, kb := range kbs {
		ids = append(ids, kb.ID)
	}
	return ids
}

// forwarded strips the hub's routing token from a federated call's arguments so
// the rest of them — limit, detail_limit, toc_path, match_id, the note id —
// reach the peer verbatim. The hub and the peer share one params type per tool,
// so nothing can be dropped by forgetting to copy a field.
func forwarded[T interface {
	model.MCPSearchParams | model.MCPSimilarParams | model.MCPNoteHTMLParams | model.MCPExpandParams
}](args T) T {
	switch p := any(&args).(type) {
	case *model.MCPSearchParams:
		p.KBID, p.KBIDs = "", nil
	case *model.MCPSimilarParams:
		p.KBID = ""
	case *model.MCPNoteHTMLParams:
		p.KBID = ""
	case *model.MCPExpandParams:
		p.KBID = ""
	}
	return args
}
