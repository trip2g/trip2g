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

	if args.KBID != "" {
		kb, rest := findFederationKB(kbs, args.KBID)
		if kb == nil {
			return federationNotConfiguredResponse(id, args.KBID)
		}
		client, err := env.FederationClient(kb.ID)
		if err != nil {
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
		if err != nil {
			return errorResponse(id, ErrCodeInternal, err.Error())
		}
		return successResponse(id, federationResultToToolResult(result))
	}

	selected := selectFederationKBs(kbs, "", args.KBIDs)
	if len(selected) == 0 {
		return federationNotConfiguredResponse(id, "")
	}
	results := fanout(ctx, env, selected, func(ctx context.Context, client model.Federation) (model.FederationResult, error) {
		return client.Search(ctx, model.FederationSearchParams{Query: args.Query})
	})
	return successResponse(id, aggregateFederationResults(results))
}

func handleFederatedSimilar(ctx context.Context, env Env, id any, argsRaw json.RawMessage) Response {
	args, errResp := unmarshalArgs[FederatedSimilarArguments](argsRaw, id, "federated_similar")
	if errResp != nil {
		return *errResp
	}
	if args.KBID == "" {
		return errorResponse(id, ErrCodeInvalidParams, "kb_id is required")
	}

	kbs, err := accessibleKBNotes(ctx, env)
	if err != nil {
		return errorResponse(id, ErrCodeInternal, err.Error())
	}
	if len(kbs) == 0 {
		return federationNotConfiguredResponse(id, args.KBID)
	}

	kb, rest := findFederationKB(kbs, args.KBID)
	if kb == nil {
		return federationNotConfiguredResponse(id, args.KBID)
	}
	client, err := env.FederationClient(kb.ID)
	if err != nil {
		return errorResponse(id, ErrCodeInternal, err.Error())
	}
	params := model.FederationSimilarParams{
		PID:    args.PID,
		NoteID: args.NoteID,
		Path:   args.Path,
		Href:   args.Href,
		Limit:  args.Limit,
	}
	var result model.FederationResult
	if rest == "" {
		result, err = client.Similar(ctx, params)
	} else {
		params.KBID = rest
		result, err = client.FederatedSimilar(ctx, params)
	}
	if err != nil {
		return errorResponse(id, ErrCodeInternal, err.Error())
	}
	return successResponse(id, federationResultToToolResult(result))
}

func handleFederatedNoteHTML(ctx context.Context, env Env, id any, argsRaw json.RawMessage) Response {
	args, errResp := unmarshalArgs[FederatedNoteHTMLArguments](argsRaw, id, "federated_note_html")
	if errResp != nil {
		return *errResp
	}
	if args.KBID == "" {
		return errorResponse(id, ErrCodeInvalidParams, "kb_id is required")
	}

	kbs, err := accessibleKBNotes(ctx, env)
	if err != nil {
		return errorResponse(id, ErrCodeInternal, err.Error())
	}
	if len(kbs) == 0 {
		return federationNotConfiguredResponse(id, args.KBID)
	}

	kb, rest := findFederationKB(kbs, args.KBID)
	if kb == nil {
		return federationNotConfiguredResponse(id, args.KBID)
	}
	client, err := env.FederationClient(kb.ID)
	if err != nil {
		return errorResponse(id, ErrCodeInternal, err.Error())
	}
	params := model.FederationNoteHTMLParams{
		PID:     args.PID,
		NoteID:  args.NoteID,
		Path:    args.Path,
		Href:    args.Href,
		MatchID: args.MatchID,
	}
	var result model.FederationResult
	if rest == "" {
		result, err = client.NoteHTML(ctx, params)
	} else {
		params.KBID = rest
		result, err = client.FederatedNoteHTML(ctx, params)
	}
	if err != nil {
		return errorResponse(id, ErrCodeInternal, err.Error())
	}
	return successResponse(id, federationResultToToolResult(result))
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
