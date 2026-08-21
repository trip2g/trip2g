package webhookutil

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"trip2g/internal/model"
	"trip2g/internal/ptr"

	ozzo "github.com/go-ozzo/ozzo-validation/v4"
)

// HashContent returns the base64url sha256 of note content — the hash format
// agents receive and send back as expected_hash (same as updateNotes CAS).
func HashContent(content []byte) string {
	sum := sha256.Sum256(content)
	return base64.URLEncoding.EncodeToString(sum[:])
}

// AgentChange kinds. An empty Kind is treated as KindWrite for backward
// compatibility with existing webhook agents that only send {path, content}.
const (
	AgentChangeKindWrite = "write" // full upsert of Content at Path.
	AgentChangeKindPatch = "patch" // surgical single-occurrence Find/Replace.
)

// AgentResponse is the expected format of a webhook agent's response body.
type AgentResponse struct {
	Status  string        `json:"status"`
	Message string        `json:"message"`
	Changes []AgentChange `json:"changes"`
	// Costs is what the run cost, as an open {unit: amount} object: the unit lives
	// in the key, so an LLM agent reports {"tokens": 5186, "steps": 2} and something
	// that bills money reports {"usd": 0.004}. trip2g stores it as given and only
	// sums per key — it has no opinion about which units exist.
	Costs map[string]float64 `json:"costs"`
	// Logs is what the agent wants an operator to see about the run. trip2g
	// stores it and hands it back; it never reads Data.
	Logs []AgentLog `json:"logs,omitempty"`
}

// AgentLog is one entry an agent reports about its run: when, how bad, a human
// line, and whatever else its author wants to carry.
//
// Data is deliberately opaque. An agent's vocabulary is its own — it may call
// tools this platform has never heard of, e.g. an MCP server declared in a role
// note — so trip2g stores Data as given and returns it unread. Only ts/level/msg
// mean anything here, and they mean the same for every agent.
type AgentLog struct {
	TS    time.Time       `json:"ts"`
	Level string          `json:"level"`
	Msg   string          `json:"msg"`
	Data  json.RawMessage `json:"data,omitempty"`
}

// AgentChange represents a single file change from an agent.
type AgentChange struct {
	Path         string  `json:"path"`
	Content      string  `json:"content"`
	ExpectedHash *string `json:"expected_hash,omitempty"`
	Find         string  `json:"find,omitempty"`
	Replace      string  `json:"replace,omitempty"`
	Kind         string  `json:"kind,omitempty"` // "" | "upsert" | "patch"
}

// IsPatch reports whether the change is a surgical find/replace patch.
func (c AgentChange) IsPatch() bool {
	return c.Kind == AgentChangeKindPatch
}

// Validate validates required fields of an AgentChange. Patch changes require
// Find (not Content); upsert/legacy changes require Content.
func (c AgentChange) Validate() error {
	if c.IsPatch() {
		return ozzo.ValidateStruct(&c,
			ozzo.Field(&c.Path, ozzo.Required),
			ozzo.Field(&c.Find, ozzo.Required),
		)
	}
	return ozzo.ValidateStruct(&c,
		ozzo.Field(&c.Path, ozzo.Required),
		ozzo.Field(&c.Content, ozzo.Required),
	)
}

// ResolveAgentChanges validates every change (write patterns, expected_hash
// CAS, patch resolution) against the current note views and returns the final
// notes to write. Any invalid change rejects the whole batch, so callers never
// apply a partial prefix.
func ResolveAgentChanges(changes []AgentChange, nvs *model.NoteViews, writePatterns []string) ([]model.RawNote, error) {
	notes := make([]model.RawNote, 0, len(changes))
	for _, change := range changes {
		// Deny-all when write_patterns is empty: a webhook delivery is always a
		// scoped context, so an empty list means "no writes permitted" rather
		// than "allow all". Also deny on no-match when non-empty.
		if len(writePatterns) == 0 || !MatchesAny(change.Path, writePatterns) {
			return nil, fmt.Errorf("path %q not allowed by write_patterns", change.Path)
		}

		var nv *model.NoteView
		if nvs != nil {
			nv = nvs.PathMap[change.Path]
		}

		// expected_hash gates the write with optimistic concurrency, matching
		// updateNotes CAS semantics: "" asserts the note is absent (create-only).
		if change.ExpectedHash != nil {
			var actualHash string
			if nv != nil {
				actualHash = HashContent(nv.Content)
			}
			if actualHash != *change.ExpectedHash {
				return nil, fmt.Errorf("expected_hash mismatch for %s: note changed since it was read", change.Path)
			}
		}

		var content string
		if change.IsPatch() {
			// Apply find/replace against the note's current content.
			// Matches updateNotes Patch semantics: find must be present exactly once.
			if nv == nil {
				return nil, fmt.Errorf("note not found for patch: %s", change.Path)
			}
			current := string(nv.Content)
			idx := strings.Index(current, change.Find)
			if idx == -1 {
				return nil, fmt.Errorf("patch find string not found in %s", change.Path)
			}
			if strings.Contains(current[idx+len(change.Find):], change.Find) {
				return nil, fmt.Errorf("patch find string is ambiguous (multiple occurrences) in %s", change.Path)
			}
			content = current[:idx] + change.Replace + current[idx+len(change.Find):]
		} else {
			// Kind=="" or "upsert": upsert with provided content.
			content = change.Content
		}

		notes = append(notes, model.RawNote{Path: change.Path, Content: content})
	}
	return notes, nil
}

// ParseAgentResponse parses a webhook response body into AgentResponse.
// Returns nil, nil if body is empty or not valid JSON (non-fatal).
func ParseAgentResponse(body []byte) (*AgentResponse, error) {
	if len(body) == 0 {
		return nil, nil
	}

	var resp AgentResponse

	err := json.Unmarshal(body, &resp)
	if err != nil {
		// Not valid JSON — acceptable, agent may return non-JSON responses.
		return nil, nil //nolint:nilerr // invalid JSON is acceptable from agents.
	}

	for i, change := range resp.Changes {
		err = change.Validate()
		if err != nil {
			return nil, fmt.Errorf("invalid change at index %d: %w", i, err)
		}
	}

	return &resp, nil
}

// MarshalCosts serializes a reported cost object for storage, keeping only
// finite numbers: the contract is {unit: amount}, and a NaN or an infinity from a
// careless agent would poison every later sum. Nothing usable yields nil, which
// leaves the delivery's costs column untouched.
func MarshalCosts(costs map[string]float64) *string {
	clean := make(map[string]float64, len(costs))
	for unit, amount := range costs {
		if unit == "" || math.IsNaN(amount) || math.IsInf(amount, 0) {
			continue
		}
		clean[unit] = amount
	}
	if len(clean) == 0 {
		return nil
	}
	data, err := json.Marshal(clean)
	if err != nil {
		return nil
	}
	return ptr.To(string(data))
}

// Agent-reported logs are agent-controlled data kept for the delivery's whole
// retention, so they are bounded on both axes: a careless or hostile agent must
// not be able to park an unbounded blob in the database.
const (
	// MaxAgentLogs is how many entries one delivery may store.
	MaxAgentLogs = 500
	// MaxAgentLogsBytes is the ceiling on the serialized result.
	MaxAgentLogsBytes = 64 * 1024
)

// MarshalLogs serializes an agent's run log for storage, trimmed to the limits
// above. Whatever it drops is replaced by a final entry saying how much went
// missing — a silently truncated log reads as a complete one. Nothing usable
// yields nil, which leaves the delivery's logs column untouched.
func MarshalLogs(logs []AgentLog) *string {
	if len(logs) == 0 {
		return nil
	}

	kept := logs
	dropped := 0
	if len(kept) > MaxAgentLogs {
		dropped = len(kept) - MaxAgentLogs
		kept = kept[:MaxAgentLogs]
	}

	// Drop from the tail until the whole thing fits. The first entries are the
	// ones that explain how a run started, so they are the ones worth keeping.
	for {
		data, err := json.Marshal(appendDropNote(kept, dropped))
		if err != nil {
			return nil
		}
		if len(data) <= MaxAgentLogsBytes || len(kept) == 0 {
			out := string(data)
			return &out
		}
		kept = kept[:len(kept)-1]
		dropped++
	}
}

// appendDropNote returns the entries plus, when anything was dropped, one more
// saying so.
func appendDropNote(kept []AgentLog, dropped int) []AgentLog {
	if dropped == 0 {
		return kept
	}
	note := AgentLog{
		Level: "warn",
		Msg:   fmt.Sprintf("%d further log entries dropped: over the %d-entry / %d-byte limit", dropped, MaxAgentLogs, MaxAgentLogsBytes),
	}
	return append(append(make([]AgentLog, 0, len(kept)+1), kept...), note)
}
