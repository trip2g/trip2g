// Package agentruntime is the Phase-1 micro-agent executor: it runs an
// instruction through an LLM tool-call loop, scoped by read/write glob
// patterns, with a non-overridable per-run token hard-cap. It reuses the
// existing webhook scope matcher and change contract (internal/webhookutil)
// so its output slots into the existing change-apply path.
package agentruntime

import "context"

// Doc is a single knowledge-base document: a path plus its content (or a
// snippet, for search results).
type Doc struct {
	Path    string
	Content string
}

// KB is the knowledge-base surface the executor needs. Implementations back it
// with a filesystem vault (FileKB), an in-memory map (tests), or — later — the
// trip2g model layer. The executor never touches a KB directly; it always goes
// through ScopedKB, which enforces read/write patterns.
type KB interface {
	// Search returns documents matching the query. Implementations decide
	// ranking; the executor only relies on Path being set.
	Search(ctx context.Context, query string) ([]Doc, error)
	// Read returns the full content of the document at path.
	Read(ctx context.Context, path string) (string, error)
	// Write upserts the document at path with the given content.
	Write(ctx context.Context, path string, content string) error
}
