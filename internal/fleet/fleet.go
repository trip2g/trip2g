package fleet

import (
	"context"
	"sync"

	"trip2g/internal/agentruntime"
)

// Fleet ties config, the trip2g client, the LLM, and the live role registry
// (keyed by url key) together. It is the HTTP handler's owner.
type Fleet struct {
	cfg    Config
	client Client
	llm    agentruntime.LLM

	mu       sync.RWMutex
	registry map[string]Role // urlKey(notePath) -> Role

	// codeRunner is the code-role executor. Defaults to agentruntime.RunCode.
	// Tests may inject a stub for deterministic testing without subprocess runs.
	codeRunner func(context.Context, agentruntime.CodeInput) (*agentruntime.Result, error)
}

// NewFleet builds a Fleet with an empty registry and the default code runner.
func NewFleet(cfg Config, client Client, llm agentruntime.LLM) *Fleet {
	return &Fleet{
		cfg:        cfg,
		client:     client,
		llm:        llm,
		registry:   map[string]Role{},
		codeRunner: agentruntime.RunCode,
	}
}

// SetRoles atomically swaps the live role registry (called after each poll).
func (f *Fleet) SetRoles(roles []Role) {
	reg := make(map[string]Role, len(roles))
	for _, r := range roles {
		reg[urlKey(r.NotePath)] = r
	}
	f.mu.Lock()
	f.registry = reg
	f.mu.Unlock()
}

func (f *Fleet) roleByKey(key string) (Role, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	r, ok := f.registry[key]
	return r, ok
}

// secretFor derives the per-role HMAC secret used to verify deliveries.
func (f *Fleet) secretFor(role Role) string {
	return deriveSecret(f.cfg.FleetSecret, f.cfg.FleetID, role.NotePath, specVer(role))
}

// clampBudget enforces the non-overridable fleet ceiling. An unset (<=0)
// frontmatter value defaults to the ceiling.
func clampBudget(want, ceiling int) int {
	if want <= 0 {
		return ceiling
	}
	return min(want, ceiling)
}
