package fleet

import (
	"fmt"
	"strconv"
	"strings"

	"trip2g/internal/webhookutil"
)

// Executor values for Role.Executor.
const (
	executorLLM  = "llm"  // default: run through agentruntime.Run (LLM tool loop)
	executorCode = "code" // deterministic: body renders to a program; stdout = write JSON
)

// Role is a parsed role note: flat frontmatter (config) + body (instruction).
type Role struct {
	NotePath       string
	Body           string
	Executor       string // "" | "llm" | "code" (default: llm)
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
	Concurrency    string   // "allow_overlap" | "skip" | "queue_one"
	ForEach        string   // "" (single run) | "changed_files" | "attached_notes"
	EnvPassthrough []string // exact env var names forwarded to code child process
	EnvPrefix      []string // env var name prefixes forwarded to code child process
}

// ParseRole builds a Role from a note path, body, and flat frontmatter meta
// (key -> raw value). List values accept JSON (["a","b"]) or YAML-flow
// ([a, b]) form; scalars are trimmed.
func ParseRole(notePath, body string, m map[string]string) (Role, error) {
	r := Role{
		NotePath:       notePath,
		Body:           body,
		Executor:       strings.TrimSpace(m["executor"]),
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
		EnvPassthrough: parseList(m["env_passthrough"]),
		EnvPrefix:      parseList(m["env_prefix"]),
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

// for_each modes. These strings double as the Jet template variable names the
// modes fan out over (forEachChangedFiles → the changed_files bag var, etc.).
const (
	forEachChangedFiles  = "changed_files"
	forEachAttachedNotes = "attached_notes"
)

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
	// Change-mode trigger sanity (only mode change reaches here): a webhook that
	// fires on no events or matches no paths silently never runs.
	if len(r.TriggerOn) == 0 {
		return fmt.Errorf("role %s: trigger_on is empty (webhook would fire on no events); set trigger_on to some of create|update|remove", r.NotePath)
	}
	if len(r.TriggerInclude) == 0 {
		return fmt.Errorf("role %s: trigger_include is empty (webhook would match no paths); set trigger_include patterns", r.NotePath)
	}
	// change_file footgun: a body referencing change_file but not fanned out per
	// changed file renders against nil, so every delivery fails. Note
	// "changed_files" (plural) does not contain the "change_file" substring.
	// For code executor roles the body IS still Jet-rendered, so the check
	// applies. However, Python variable names like `change_file = ...` would
	// be false positives; a future improvement could scope this more narrowly.
	if strings.Contains(r.Body, "change_file") && r.ForEach != forEachChangedFiles {
		return fmt.Errorf(
			"role %s: body references change_file but for_each is not changed_files "+
				"(renders against nil); set for_each: changed_files",
			r.NotePath,
		)
	}
	switch r.Concurrency {
	case "", "allow_overlap", "skip", "queue_one":
	default:
		return fmt.Errorf("role %s: concurrency must be allow_overlap|skip|queue_one, got %q", r.NotePath, r.Concurrency)
	}
	switch r.ForEach {
	case "", forEachChangedFiles, forEachAttachedNotes:
	default:
		return fmt.Errorf("role %s: for_each must be changed_files|attached_notes, got %q", r.NotePath, r.ForEach)
	}
	if r.TimeoutSeconds < 0 {
		return fmt.Errorf("role %s: timeout_seconds must be >= 0, got %d", r.NotePath, r.TimeoutSeconds)
	}
	// Executor-specific validation.
	switch r.Executor {
	case "", executorLLM:
		// LLM path (default): validate the role-declared tool allowlist.
		for _, t := range r.Tools {
			if !contains(offered, t) {
				return fmt.Errorf("role %s: tool %q not offered by this fleet", r.NotePath, t)
			}
		}
	case executorCode:
		// Code path: require a write scope and a resolvable fenced program block.
		if len(r.WritePatterns) == 0 {
			return fmt.Errorf("role %s: executor:code requires at least one write_patterns entry", r.NotePath)
		}
		lang, found := roleFenceLang(r.Body)
		if !found {
			return fmt.Errorf("role %s: executor:code body must contain a fenced code block (```lang...```)", r.NotePath)
		}
		if !resolveFenceLang(lang) {
			return fmt.Errorf("role %s: executor:code fence language %q not supported (supported: python, bash, node)", r.NotePath, lang)
		}
		// env_prefix safety: an empty-string prefix would match every env var,
		// defeating the deny-by-default secret-scrub guarantee.
		for _, p := range r.EnvPrefix {
			if p == "" {
				return fmt.Errorf("role %s: env_prefix must not contain an empty entry (would match all env vars)", r.NotePath)
			}
		}
	default:
		return fmt.Errorf("role %s: executor must be llm|code, got %q", r.NotePath, r.Executor)
	}
	return nil
}

// roleFenceLang scans body for the first fenced code block and returns its
// language tag. Returns ("", false) when no complete block is found.
func roleFenceLang(body string) (string, bool) {
	idx := strings.Index(body, "```")
	if idx == -1 {
		return "", false
	}
	rest := body[idx+3:]
	nl := strings.IndexByte(rest, '\n')
	if nl == -1 {
		return "", false
	}
	langStr := strings.TrimSpace(rest[:nl])
	end := strings.Index(rest[nl+1:], "```")
	return langStr, end != -1
}

// resolveFenceLang reports whether a fence language tag maps to a supported
// code executor program (python, bash, or node).
func resolveFenceLang(lang string) bool {
	switch strings.ToLower(lang) {
	case "python", "py", "bash", "sh", "js", "javascript", "node":
		return true
	}
	return false
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
