package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"slices"

	graphmodel "trip2g/internal/graph/model"
	"trip2g/internal/metrics"
	"trip2g/internal/model"
)

type mcpMetricsContextKey struct{}

// ContextWithMetrics stores MCP metrics in ctx so tool handlers can record
// tool-specific observations (fan-out width, result counts, dynamic tools).
func ContextWithMetrics(ctx context.Context, m *metrics.MCPMetrics) context.Context {
	return context.WithValue(ctx, mcpMetricsContextKey{}, m)
}

// metricsFromContext returns the MCP metrics from ctx, or nil (all record
// methods are nil-safe).
func metricsFromContext(ctx context.Context) *metrics.MCPMetrics {
	m, _ := ctx.Value(mcpMetricsContextKey{}).(*metrics.MCPMetrics)
	return m
}

// recordRequestMetrics records the per-request metrics after Resolve.
func recordRequestMetrics(ctx context.Context, m *metrics.MCPMetrics, req Request, hasUserToken bool, resp Response, seconds float64) {
	if m == nil {
		return
	}

	auth := authKind(ctx, hasUserToken)
	method := methodLabel(req.Method)

	tool := ""
	if req.Method == mcpMethodToolsCall {
		var params CallToolParams
		if json.Unmarshal(req.Params, &params) == nil {
			tool = toolLabel(params.Name, resp)
		}
	}

	status := "ok"
	if resp.Error != nil {
		status = "error"
	}

	m.RecordMCPRequest(method, tool, auth, status, seconds)
	if fedAuth, ok := federationAuthFromContext(ctx); ok {
		inboundTool := tool
		if inboundTool == "" {
			inboundTool = method
		}
		m.RecordFederatedInbound(fedAuth.KID, inboundTool)
	}
	if req.Method == mcpMethodToolsList {
		m.RecordToolsList(auth)
	}
	if req.Method == mcpMethodToolsCall && resp.Error != nil {
		m.RecordToolError(tool, errorReason(resp.Error.Code))
	}
}

// recordRejectedRequest counts a request rejected before Resolve (depth limit,
// auth failures) so error and abusive traffic shows up in the request counters.
// auth is the attempted auth kind; the tool label stays empty since the call
// never reached tool dispatch.
func recordRejectedRequest(m *metrics.MCPMetrics, req Request, auth string, seconds float64) {
	m.RecordMCPRequest(methodLabel(req.Method), "", auth, "error", seconds)
}

// DynamicToolCount returns how many notes expose a dynamic (mcp_method) tool,
// excluding names shadowed by built-in tools. The app wires it as the
// scrape-time source of the trip2g_mcp_dynamic_tools_registered gauge.
func DynamicToolCount(nvs *model.NoteViews) int {
	if nvs == nil {
		return 0
	}
	n := 0
	for _, note := range nvs.List {
		if note.MCPMethod != "" && !reservedMCPTools[note.MCPMethod] {
			n++
		}
	}
	return n
}

// Auth kinds for the auth metric label.
const (
	authToken      = "token"
	authAPIKey     = "api_key"
	authFederation = "federation"
	authAnonymous  = "anonymous"
)

// authKind classifies the request auth for metric labels.
func authKind(ctx context.Context, hasUserToken bool) string {
	switch {
	case hasUserToken:
		return authToken
	case mcpAPIKeyAuthed(ctx):
		return authAPIKey
	default:
		if _, ok := federationAuthFromContext(ctx); ok {
			return authFederation
		}
		return authAnonymous
	}
}

// otherLabel is the bounded fallback for free-form client input in labels.
const otherLabel = "other"

// dynamicLabel is the fixed label for note-registered (mcp_method) tools:
// frontmatter names are author-controlled and unbounded, so they never become
// label values.
const dynamicLabel = "dynamic"

// methodLabel bounds the method label to the known JSON-RPC method set.
func methodLabel(method string) string {
	switch method {
	case MCPMethodInitialize, "notifications/initialized", mcpMethodToolsList, mcpMethodToolsCall:
		return method
	default:
		return otherLabel
	}
}

// toolLabel bounds the tool label: built-in tools keep their name (fixed set);
// a name Resolve rejected as unknown maps to "other"; anything else Resolve
// accepted is a note-registered dynamic tool and maps to "dynamic".
func toolLabel(name string, resp Response) string {
	if reservedMCPTools[name] {
		return name
	}
	if resp.Error != nil && resp.Error.Code == ErrCodeMethodNotFound {
		return otherLabel
	}
	return dynamicLabel
}

// errorReason maps a JSON-RPC error code to a bounded reason label.
func errorReason(code int) string {
	switch code {
	case ErrCodeInvalidParams:
		return "invalid_params"
	case ErrCodeMethodNotFound:
		return "not_found"
	case ErrCodeInternal:
		return "internal"
	default:
		return otherLabel
	}
}

// federatedStatus maps an outbound federated call error to ok|error|timeout.
func federatedStatus(err error) string {
	if err == nil {
		return "ok"
	}
	var timeout interface{ Timeout() bool }
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &timeout) && timeout.Timeout()) {
		return "timeout"
	}
	return "error"
}

// noSubgraphLabel marks a note a peer could read without any of its key's
// subgraphs: a free note, a sign-in note, or one outside every subgraph.
const noSubgraphLabel = "none"

// grantingSubgraph names the subgraph through which a key reads note: the
// note's first subgraph the key is scoped to. Only names from the key's own
// scope become label values, so the set stays bounded by configured subgraphs.
func grantingSubgraph(note *model.NoteView, allowed []string) string {
	for _, name := range note.SubgraphNames {
		if slices.Contains(allowed, name) {
			return name
		}
	}
	return noSubgraphLabel
}

// recordFederatedNoteServed counts a note or section handed to a federated
// peer. Requests not authenticated by an inbound key record nothing.
func recordFederatedNoteServed(ctx context.Context, note *model.NoteView) {
	auth, ok := federationAuthFromContext(ctx)
	if !ok {
		return
	}
	metricsFromContext(ctx).RecordFederatedNoteServed(auth.KID, grantingSubgraph(note, auth.AllowedSubgraphs))
}

// recordFederatedResultsServed counts each search result returned to a
// federated peer under the subgraph that granted it.
func recordFederatedResultsServed(ctx context.Context, notes []*model.NoteView) {
	auth, ok := federationAuthFromContext(ctx)
	if !ok {
		return
	}
	m := metricsFromContext(ctx)
	for _, note := range notes {
		m.RecordFederatedResultServed(auth.KID, grantingSubgraph(note, auth.AllowedSubgraphs))
	}
}

// servedSearchNotes returns the notes behind the first n results, in the order
// buildSearchPayload turns them into the answer.
func servedSearchNotes(results []model.SearchResult, n int) []*model.NoteView {
	notes := make([]*model.NoteView, 0, n)
	for _, r := range results {
		if len(notes) >= n {
			break
		}
		if r.NoteView != nil {
			notes = append(notes, r.NoteView)
		}
	}
	return notes
}

// similarNotes returns the notes a similar answer lists.
func similarNotes(results []graphmodel.SimilarNote) []*model.NoteView {
	notes := make([]*model.NoteView, 0, len(results))
	for _, r := range results {
		if r.Note != nil && r.Note.NoteView != nil {
			notes = append(notes, r.Note.NoteView)
		}
	}
	return notes
}
