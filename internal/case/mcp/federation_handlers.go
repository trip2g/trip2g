package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"trip2g/internal/model"
)

func handleFederatedSearch(ctx context.Context, env Env, id any, argsRaw json.RawMessage) Response {
	args, errResp := unmarshalArgs[FederatedSearchArguments](argsRaw, id, "federated_search")
	if errResp != nil {
		return *errResp
	}
	if args.Query == "" {
		return errorResponse(id, ErrCodeInvalidParams, "query is required")
	}

	kbs, err := accessibleKBNotes(ctx, env)
	if err != nil {
		return errorResponse(id, ErrCodeInternal, err.Error())
	}
	if len(kbs) == 0 {
		return federationNotConfiguredResponse(id, args.KBID)
	}

	if args.KBID == "" {
		selected := selectFederationKBs(kbs, "", args.KBIDs)
		if len(selected) == 0 {
			return federationNotConfiguredResponse(id, "")
		}
		results := fanout(ctx, env, selected, func(ctx context.Context, client model.Federation) (model.FederationResult, error) {
			return client.Search(ctx, model.FederationSearchParams{Query: args.Query})
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
		return successResponse(id, aggregateFederationResults(results))
	}

	kb, rest := findFederationKB(kbs, args.KBID)
	if kb == nil {
		return federationNotConfiguredResponse(id, args.KBID)
	}
	var client model.Federation
	client, err = env.FederationClient(ctx, kb.ID)
	if err != nil {
		// A client that can't be built is still a failed outbound attempt,
		// same as on the fan-out path.
		metricsFromContext(ctx).RecordFederatedRequest(federatedStatus(err))
		return errorResponse(id, ErrCodeInternal, err.Error())
	}
	params := model.FederationSearchParams{Query: args.Query}
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
	kbs, err := accessibleKBNotes(ctx, env)
	if err != nil {
		return errorResponse(id, ErrCodeInternal, err.Error())
	}
	if len(kbs) == 0 {
		return federationNotConfiguredResponse(id, kbID)
	}
	kb, rest := findFederationKB(kbs, kbID)
	if kb == nil {
		return federationNotConfiguredResponse(id, kbID)
	}
	client, err := env.FederationClient(ctx, kb.ID)
	if err != nil {
		// A client that can't be built is still a failed outbound attempt,
		// same as on the fan-out path.
		metricsFromContext(ctx).RecordFederatedRequest(federatedStatus(err))
		return errorResponse(id, ErrCodeInternal, err.Error())
	}
	result, err := call(client, rest)
	metricsFromContext(ctx).RecordFederatedRequest(federatedStatus(err))
	if err != nil {
		return errorResponse(id, ErrCodeInternal, err.Error())
	}
	return successResponse(id, federationResultToToolResult(result))
}

func handleFederatedSimilar(ctx context.Context, env Env, id any, argsRaw json.RawMessage) Response {
	args, errResp := unmarshalArgs[FederatedSimilarArguments](argsRaw, id, "federated_similar")
	if errResp != nil {
		return *errResp
	}
	if args.KBID == "" {
		return errorResponse(id, ErrCodeInvalidParams, "kb_id is required")
	}
	params := model.FederationSimilarParams{
		PID:    args.PID,
		NoteID: args.NoteID,
		Path:   args.Path,
		Href:   args.Href,
		Limit:  args.Limit,
	}
	return callFederatedSingleKB(ctx, env, id, args.KBID, func(client model.Federation, rest string) (model.FederationResult, error) {
		if rest == "" {
			return client.Similar(ctx, params)
		}
		params.KBID = rest
		return client.FederatedSimilar(ctx, params)
	})
}

func handleFederatedNoteHTML(ctx context.Context, env Env, id any, argsRaw json.RawMessage) Response {
	args, errResp := unmarshalArgs[FederatedNoteHTMLArguments](argsRaw, id, "federated_note_html")
	if errResp != nil {
		return *errResp
	}
	if args.KBID == "" {
		return errorResponse(id, ErrCodeInvalidParams, "kb_id is required")
	}
	params := model.FederationNoteHTMLParams{
		PID:     args.PID,
		NoteID:  args.NoteID,
		Path:    args.Path,
		Href:    args.Href,
		MatchID: args.MatchID,
	}
	return callFederatedSingleKB(ctx, env, id, args.KBID, func(client model.Federation, rest string) (model.FederationResult, error) {
		if rest == "" {
			return client.NoteHTML(ctx, params)
		}
		params.KBID = rest
		return client.FederatedNoteHTML(ctx, params)
	})
}

func handleFederatedExpand(ctx context.Context, env Env, id any, argsRaw json.RawMessage) Response {
	args, errResp := unmarshalArgs[FederatedExpandArguments](argsRaw, id, "federated_expand")
	if errResp != nil {
		return *errResp
	}
	if args.KBID == "" {
		return errorResponse(id, ErrCodeInvalidParams, "kb_id is required")
	}
	params := model.FederationExpandParams{
		PID:     args.PID,
		NoteID:  args.NoteID,
		Path:    args.Path,
		Href:    args.Href,
		TocPath: args.TocPath,
	}
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
		return client.GraphQLRequest(ctx, model.FederationGraphQLParams{KBID: rest, Query: args.Query, Variables: args.Variables})
	})
}

func federationNotConfiguredResponse(id any, kbID string) Response {
	message := "Federation is not configured for this hub. No KB-notes were found. To enable federation, create a note with mcp_federation_kb_url frontmatter pointing to another MCP endpoint."
	status := "federation_not_configured"
	if kbID != "" {
		message = fmt.Sprintf("Federation is not configured for kb_id %q.", kbID)
	}
	payload := FederationStatusPayload{
		Status:  status,
		KBID:    kbID,
		Message: message,
	}
	return successResponse(id, structuredToolResult(message, payload))
}
