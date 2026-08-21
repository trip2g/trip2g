package model

// One params type per MCP tool, shared by all three places a tool call passes
// through: the local handler decoding a client's arguments, the hub decoding a
// federated_* call, and the wire format sent to a peer. They used to be three
// parallel struct families, which silently dropped any field one of them was
// missing (limit, detail_limit, toc_path) and disagreed on the type of note_id.
//
// Deliberately kept out of federation.go: easyjson generates over that file,
// and its decoder reports "parse error near offset 11" where encoding/json
// says which field had the wrong type — and these errors are read by models
// correcting their own tool calls.

type MCPSearchParams struct {
	Query       string   `json:"query"`
	KBID        string   `json:"kb_id,omitempty"`
	KBIDs       []string `json:"kb_ids,omitempty"`
	Limit       int      `json:"limit,omitempty"`
	DetailLimit int      `json:"detail_limit,omitempty"`

	// Rerank is the caller's explicit preference for the second-stage
	// cross-encoder; nil means "no preference" and the instance default
	// decides. It rides along through forwarded() on federated_search, so a
	// peer applies it against ITS OWN reranker config — reranking happens
	// where the corpus is, which is the only place the passages exist.
	Rerank *bool `json:"rerank,omitempty"`
}

type MCPSimilarParams struct {
	KBID   string `json:"kb_id,omitempty"`
	PID    PID    `json:"pid,omitzero"`
	NoteID PID    `json:"note_id,omitzero"`
	Path   string `json:"path,omitempty"`
	Href   string `json:"href,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type MCPInstructionsParams struct {
	KBID string `json:"kb_id,omitempty"`
}

type MCPGraphQLParams struct {
	KBID      string         `json:"kb_id,omitempty"`
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type MCPNoteHTMLParams struct {
	KBID         string   `json:"kb_id,omitempty"`
	PID          PID      `json:"pid,omitzero"`
	NoteID       PID      `json:"note_id,omitzero"`
	Path         string   `json:"path,omitempty"`
	Href         string   `json:"href,omitempty"`
	MatchID      string   `json:"match_id,omitempty"`
	ContextWords int      `json:"context_words,omitempty"`
	TocPath      []string `json:"toc_path,omitempty"`
}

type MCPExpandParams struct {
	KBID    string   `json:"kb_id,omitempty"`
	PID     PID      `json:"pid,omitzero"`
	NoteID  PID      `json:"note_id,omitzero"`
	Path    string   `json:"path,omitempty"`
	Href    string   `json:"href,omitempty"`
	TocPath []string `json:"toc_path,omitempty"`
}
