package mcp

import (
	"context"
	"encoding/json"
	"errors"

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
func recordRequestMetrics(ctx context.Context, m *metrics.MCPMetrics, env Env, req Request, hasUserToken bool, resp Response, seconds float64) {
	if m == nil {
		return
	}

	auth := authKind(ctx, hasUserToken)
	method := methodLabel(req.Method)

	tool := ""
	if req.Method == mcpMethodToolsCall {
		var params CallToolParams
		if json.Unmarshal(req.Params, &params) == nil {
			tool = toolLabel(env, params.Name)
		}
	}

	status := "ok"
	if resp.Error != nil {
		status = "error"
	}

	m.RecordMCPRequest(method, tool, auth, status, seconds)
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
// excluding names shadowed by built-in tools. The app refreshes the
// trip2g_mcp_dynamic_tools_registered gauge with it on every notes reload.
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

// methodLabel bounds the method label to the known JSON-RPC method set.
func methodLabel(method string) string {
	switch method {
	case MCPMethodInitialize, "notifications/initialized", mcpMethodToolsList, mcpMethodToolsCall:
		return method
	default:
		return otherLabel
	}
}

// toolLabel bounds the tool label: built-in tools and note-registered dynamic
// tools keep their name, anything else (arbitrary client input) becomes "other".
func toolLabel(env Env, name string) string {
	if reservedMCPTools[name] {
		return name
	}
	if nvs := env.LatestNoteViews(); nvs != nil {
		for _, note := range nvs.List {
			if note.MCPMethod == name {
				return name
			}
		}
	}
	return otherLabel
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
