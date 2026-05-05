package model

import (
	"net/url"
	"strconv"
	"strings"
)

type MCPFederationNote struct {
	Note     *NoteView
	URL      string
	ID       string
	MaxDepth int
}

func (n *NoteView) extractMCPFederationFields() {
	url, ok := n.RawMeta["mcp_federation_kb_url"].(string)
	if !ok {
		return
	}

	n.MCPFederationKBURL = strings.TrimSpace(url)
	if id, idOk := n.RawMeta["mcp_federation_kb_id"].(string); idOk {
		n.MCPFederationKBID = strings.TrimSpace(id)
	}
	n.MCPFederationKBMaxDepth = parseMCPFederationKBMaxDepth(n.RawMeta["mcp_federation_kb_max_depth"])
}

func parseMCPFederationKBMaxDepth(raw any) int {
	switch value := raw.(type) {
	case int:
		if value > 0 {
			return value
		}
	case int64:
		if value > 0 {
			return int(value)
		}
	case float64:
		if value > 0 {
			return int(value)
		}
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil && parsed > 0 {
			return parsed
		}
	}
	return 0
}

func newMCPFederationNote(n *NoteView) *MCPFederationNote {
	return NewMCPFederationNote(n)
}

func NewMCPFederationNote(n *NoteView) *MCPFederationNote {
	if n == nil || n.MCPFederationKBURL == "" {
		return nil
	}

	id := n.MCPFederationKBID
	if id == "" {
		id = hostnameFromURL(n.MCPFederationKBURL)
	}

	return &MCPFederationNote{
		Note:     n,
		URL:      n.MCPFederationKBURL,
		ID:       id,
		MaxDepth: n.MCPFederationKBMaxDepth,
	}
}

func hostnameFromURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	return strings.ToLower(parsed.Hostname())
}

func (nv *NoteViews) ExtractMCPFederationNotes() {
	nv.MCPFederationNotes = nv.MCPFederationNotes[:0]
	seen := make(map[int64]struct{})
	for _, note := range nv.Map {
		if _, ok := seen[note.PathID]; ok {
			continue
		}
		seen[note.PathID] = struct{}{}

		kb := newMCPFederationNote(note)
		if kb == nil {
			continue
		}
		nv.MCPFederationNotes = append(nv.MCPFederationNotes, kb)
	}
}
