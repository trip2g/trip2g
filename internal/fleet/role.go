package fleet

import (
	"fmt"
	"strconv"
	"strings"

	"trip2g/internal/webhookutil"
)

// Role is a parsed role note: flat frontmatter (config) + body (instruction).
type Role struct {
	NotePath       string
	Body           string
	Model          string
	Tools          []string
	ReadPatterns   []string
	WritePatterns  []string
	MaxTokens      int
	MaxSteps       int
	TimeoutSeconds int    // 0 = unset (defaults to defaultTimeoutSeconds)
	Mode           string // "change" | "cron" | "both"
	TriggerInclude []string
	TriggerExclude []string
	TriggerOn      []string // create | update | remove
	CronSchedule   string
	AttachNotes    []string
	MaxDepth       int
	Concurrency    string // "allow_overlap" | "skip" | "queue_one"
	ForEach        string // "" (single run) | "changed_files" | "attached_notes"
}

// ParseRole builds a Role from a note path, body, and flat frontmatter meta
// (key -> raw value). List values accept JSON (["a","b"]) or YAML-flow
// ([a, b]) form; scalars are trimmed.
func ParseRole(notePath, body string, m map[string]string) (Role, error) {
	r := Role{
		NotePath:       notePath,
		Body:           body,
		Model:          strings.TrimSpace(m["model"]),
		Tools:          parseList(m["tools"]),
		ReadPatterns:   parseList(m["read_patterns"]),
		WritePatterns:  parseList(m["write_patterns"]),
		Mode:           strings.TrimSpace(m["mode"]),
		TriggerInclude: parseList(m["trigger_include"]),
		TriggerExclude: parseList(m["trigger_exclude"]),
		TriggerOn:      parseList(m["trigger_on"]),
		CronSchedule:   strings.TrimSpace(m["cron_schedule"]),
		AttachNotes:    parseList(m["attach_notes"]),
		Concurrency:    strings.TrimSpace(m["concurrency"]),
		ForEach:        strings.TrimSpace(m["for_each"]),
	}
	var err error
	if r.MaxTokens, err = parseIntOpt(m["max_tokens"]); err != nil {
		return Role{}, fmt.Errorf("max_tokens: %w", err)
	}
	if r.MaxSteps, err = parseIntOpt(m["max_steps"]); err != nil {
		return Role{}, fmt.Errorf("max_steps: %w", err)
	}
	if r.MaxDepth, err = parseIntOpt(m["max_depth"]); err != nil {
		return Role{}, fmt.Errorf("max_depth: %w", err)
	}
	if r.TimeoutSeconds, err = parseIntOpt(m["timeout_seconds"]); err != nil {
		return Role{}, fmt.Errorf("timeout_seconds: %w", err)
	}
	return r, nil
}

// defaultTimeoutSeconds is the agent-run timeout applied when a role omits
// timeout_seconds. It is deliberately generous: LLM agent runs routinely exceed
// trip2g's 60s change-webhook default, so the reconciler registers a webhook
// timeoutSeconds long enough for trip2g to wait for the run instead of closing
// the delivery connection mid-run (which would cancel the agent and lose the
// write-back).
const defaultTimeoutSeconds = 300

// EffectiveTimeoutSeconds returns the role's run timeout in seconds, applying
// defaultTimeoutSeconds when timeout_seconds is unset (0).
func (r Role) EffectiveTimeoutSeconds() int {
	if r.TimeoutSeconds <= 0 {
		return defaultTimeoutSeconds
	}
	return r.TimeoutSeconds
}

// Validate fails fast on misconfiguration discovered at poll time, before any
// webhook is registered. Tools must be a subset of the fleet's offered set.
func (r Role) Validate(offered []string) error {
	switch r.Mode {
	case "change":
		// supported
	case "cron", "both":
		return fmt.Errorf("role %s: mode %q is not yet supported by this fleet (cron-mode roles are not yet supported by this fleet)", r.NotePath, r.Mode)
	default:
		return fmt.Errorf("role %s: mode must be change|cron|both, got %q", r.NotePath, r.Mode)
	}
	switch r.Concurrency {
	case "", "allow_overlap", "skip", "queue_one":
	default:
		return fmt.Errorf("role %s: concurrency must be allow_overlap|skip|queue_one, got %q", r.NotePath, r.Concurrency)
	}
	switch r.ForEach {
	case "", "changed_files", "attached_notes":
	default:
		return fmt.Errorf("role %s: for_each must be changed_files|attached_notes, got %q", r.NotePath, r.ForEach)
	}
	if r.TimeoutSeconds < 0 {
		return fmt.Errorf("role %s: timeout_seconds must be >= 0, got %d", r.NotePath, r.TimeoutSeconds)
	}
	for _, t := range r.Tools {
		if !contains(offered, t) {
			return fmt.Errorf("role %s: tool %q not offered by this fleet", r.NotePath, t)
		}
	}
	return nil
}

func contains(set []string, v string) bool {
	for _, s := range set {
		if s == v {
			return true
		}
	}
	return false
}

// parseList accepts JSON arrays or YAML-flow arrays of strings.
func parseList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if parsed, err := webhookutil.ParseJSONStringArray(raw); err == nil {
		return parsed
	}
	raw = strings.TrimPrefix(raw, "[")
	raw = strings.TrimSuffix(raw, "]")
	var out []string
	// trip2g's note meta.raw renders a YAML list as space-joined inside brackets
	// (e.g. "[read_note write_note]"), so split on whitespace as well as commas.
	// Safe for tools and globs (none contain internal spaces).
	for _, part := range strings.Fields(strings.ReplaceAll(raw, ",", " ")) {
		v := strings.Trim(part, `"'`)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func parseIntOpt(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	return strconv.Atoi(raw)
}
