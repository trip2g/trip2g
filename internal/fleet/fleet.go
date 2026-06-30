package fleet

import (
	"context"
	"net/http"
	"sync"

	"trip2g/internal/agentruntime"
)

// Fleet ties config, the trip2g HTTP client, the LLM, and the live role
// registry (keyed by url key) together. It is the HTTP handler's owner.
// hc is the HTTP client used to build per-delivery scoped graphql clients.
type Fleet struct {
	cfg Config
	hc  *http.Client
	llm agentruntime.LLM

	mu       sync.RWMutex
	registry map[string]Role // urlKey(notePath) -> Role

	// codeRunner is the code-role executor. Defaults to agentruntime.RunCode.
	// Tests may inject a stub for deterministic testing without subprocess runs.
	codeRunner func(context.Context, agentruntime.CodeInput) (*agentruntime.Result, error)
}

// NewFleet builds a Fleet with an empty registry and the default code runner.
// hc is the HTTP client used for per-delivery scoped KB requests; nil uses
// http.DefaultClient.
func NewFleet(cfg Config, hc *http.Client, llm agentruntime.LLM) *Fleet {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &Fleet{
		cfg:        cfg,
		hc:         hc,
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

// secretFor derives the per-role HMAC secret used to verify change deliveries.
func (f *Fleet) secretFor(role Role) string {
	return deriveSecret(f.cfg.FleetSecret, f.cfg.FleetID, role.NotePath, specVer(role))
}

// cronSecretFor derives the per-role HMAC secret used to verify cron deliveries.
// It uses cronSpecVer so change-webhook and cron-webhook secrets are distinct.
func (f *Fleet) cronSecretFor(role Role) string {
	return deriveSecret(f.cfg.FleetSecret, f.cfg.FleetID, role.NotePath, cronSpecVer(role))
}

// clampBudget enforces the non-overridable fleet ceiling. An unset (<=0)
// frontmatter value defaults to the ceiling.
func clampBudget(want, ceiling int) int {
	if want <= 0 {
		return ceiling
	}
	return min(want, ceiling)
}
