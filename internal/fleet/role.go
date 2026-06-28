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
	Mode           string // "change" | "cron" | "both"
	TriggerInclude []string
	TriggerExclude []string
	TriggerOn      []string // create | update | remove
	CronSchedule   string
	AttachNotes    []string
	MaxDepth       int
	Concurrency    string // "allow_overlap" | "skip" | "queue_one"
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
	return r, nil
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
	for _, part := range strings.Split(raw, ",") {
		v := strings.Trim(strings.TrimSpace(part), `"'`)
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
