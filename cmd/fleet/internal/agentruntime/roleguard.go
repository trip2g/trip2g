package agentruntime

import (
	"errors"
	"strings"

	"gopkg.in/yaml.v3"
)

// ErrRoleAuthoringDenied is returned when a write or patch would create a role
// note, or would edit an existing one.
//
// A role note is a note whose frontmatter carries fleet_id. It is the marker
// that makes a note executable: DiscoverRoles skips a note with an empty
// fleet_id ("untagged roles are never claimed") and one tagged for a different
// fleet, so nothing without it ever runs. A role also declares its OWN
// write_patterns, tools and model, which is why authoring one is privilege
// escalation rather than an ordinary write: a role confined to transcripts/**
// could otherwise mint a successor with write_patterns ["**"], and the
// reconciler would pick it up out of band, outside any delivery-chain depth
// limit. The realistic trigger is not a malicious operator but prompt injection
// through note content, which reaches an LLM role as changed_files content.
//
// The guard keys on the marker, never on the path, so role notes stay free to
// live anywhere in the vault.
var ErrRoleAuthoringDenied = errors.New("write denied: agents may not author or edit role notes (fleet_id in frontmatter)")

// ErrRoleGuardUnverifiable is returned when a patch cannot be checked because
// the note could not be read. It fails closed, and is kept distinct from
// ErrRoleAuthoringDenied so an infrastructure failure is never reported as an
// accusation of role authoring.
var ErrRoleGuardUnverifiable = errors.New("write denied: cannot verify the patched note is not a role note")

const frontmatterDelimiter = "---"

// declaresRole reports whether content's YAML frontmatter declares a top-level
// fleet_id key, with or without a value: an empty fleet_id cannot run, but it
// is one keystroke from one that can, and nothing legitimate writes it.
//
// Only a top-level key counts, matching how fleet reads role config out of
// trip2g's flat note meta. The parse is deliberately more permissive than
// trip2g's about what counts as frontmatter (leading blank lines, CRLF,
// trailing spaces on the delimiters) — erring towards seeing a role that
// trip2g would not is a false denial, which is loud and recoverable, while the
// opposite is a silent hole.
func declaresRole(content string) bool {
	body := strings.ReplaceAll(content, "\r\n", "\n")
	body = strings.TrimPrefix(body, "\uFEFF")
	body = strings.TrimLeft(body, "\n")
	lines := strings.Split(body, "\n")
	if strings.TrimRight(lines[0], " \t") != frontmatterDelimiter {
		return false
	}
	front, ok := untilClosingDelimiter(lines[1:])
	if !ok {
		return false
	}
	var meta map[string]any
	if err := yaml.Unmarshal([]byte(front), &meta); err != nil {
		// Unparseable frontmatter yields no meta in trip2g either, so the note
		// cannot become a role through it.
		return false
	}
	_, found := meta["fleet_id"]
	return found
}

// untilClosingDelimiter returns everything before the closing "---" line and
// whether one was found. Frontmatter without a terminator is not frontmatter.
func untilClosingDelimiter(lines []string) (string, bool) {
	var out []string
	for _, line := range lines {
		if strings.TrimRight(line, " \t") == frontmatterDelimiter {
			return strings.Join(out, "\n"), true
		}
		out = append(out, line)
	}
	return "", false
}

// applyPatchPreview returns what the note would contain after a find/replace,
// mirroring trip2g's server-side semantics exactly (updatenotes.applyPatch):
// the match must be present and unique, otherwise the patch is rejected there
// and the content is unchanged. Preview-only — the real edit stays atomic and
// server-side; this never writes.
func applyPatchPreview(content, find, replace string) string {
	idx := strings.Index(content, find)
	if idx < 0 {
		return content
	}
	if strings.Contains(content[idx+len(find):], find) {
		return content // ambiguous: trip2g reports PatchNotFound
	}
	return content[:idx] + replace + content[idx+len(find):]
}
