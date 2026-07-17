// Package fleet is the trip2g agent host (fleet-as-executor): a standalone
// daemon that discovers role notes, reconciles trip2g webhooks to itself,
// receives deliveries, runs agentruntime.Run against a per-delivery scoped
// trip2g token, and writes results back. trip2g stays a dumb event source.
package fleet

import "time"

// Config is the fleet's machine-level configuration (env + flags). The role
// note's frontmatter configures the agent; this configures the host. The fleet
// ceiling (TokenCeiling/StepCeiling) is the non-overridable floor: the
// effective budget is min(frontmatter, ceiling).
type Config struct {
	FleetID                string        // required identity: reconcile marker prefix "fleet:<FleetID>:", role fleet_id partition key, and /_fleet/<sha256("fleet:"+id)>/webhook delivery path
	ListenAddr             string        // ":9090"
	CallbackURL            string        // trip2g-reachable base; webhook url = CallbackURL + "/_fleet/<h>/webhook/" + urlKey(path)
	Trip2gBaseURL          string        // e.g. "http://localhost:20081"
	AdminAPIKey            string        // DEPRECATED/unused: legacy full-admin X-Api-Key (admin lane now authenticates via HAT)
	JWTSecret              string        // shared user-token/JWT secret used to mint admin HATs (= trip2g UserToken.Secret)
	AdminEmail             string        // admin identity the fleet self-provisions via HAT (default "fleet@local")
	FleetSecret            string        // per-role HMAC secret seed
	LLMBaseURL             string        // OpenAI-compatible base URL (fleet-local, NOT a trip2g secret)
	LLMAPIKey              string        // fleet-local LLM credential
	ExecBaseURL            string        // OpenAI-compatible endpoint the exec tool routes code to (codellm); empty = exec disabled
	ExecAPIKey             string        // credential for ExecBaseURL
	DefaultModel           string        // fallback when a role omits model
	TokenCeiling           int           // non-overridable per-run token cap
	StepCeiling            int           // non-overridable per-run step cap
	AgentsFolder           string        // e.g. "roles/" -> notePaths like "roles/%"
	OfferedTools           []string      // a role's tools must be a subset of these
	PollInterval           time.Duration // discovery/reconcile poll cadence
	ShutdownGrace          time.Duration // max time to drain in-flight runs on shutdown
	KeepWebhooksOnShutdown bool          // skip webhook deregister on shutdown (rolling deploys; trip2g retains + retries)
	LogLevel               string        // zerolog level: debug|info|warn|error
}
