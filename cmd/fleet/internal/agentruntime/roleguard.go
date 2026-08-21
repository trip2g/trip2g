package agentruntime

import (
	"errors"
	"strings"

	yaml "gopkg.in/yaml.v2"
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

// declaresRole reports whether content's YAML frontmatter declares a top-level
// fleet_id key, with or without a value: an empty fleet_id cannot run, but it
// is one keystroke from one that can, and nothing legitimate writes it.
//
// The frontmatter rules mirror goldmark-meta, which is what actually produces a
// note's meta in trip2g (internal/mdloader -> NoteView.RawMeta -> the GraphQL
// meta field fleet's DiscoverRoles reads fleet_id from). Mirroring matters more
// than elegance here: every input goldmark parses as frontmatter but this
// function does not is a silent hole in the guard, which is why
// TestDeclaresRoleMatchesGoldmark diffs the two directly. The YAML library is
// v2 for the same reason, not by preference: v3 rejects duplicate keys that v2
// accepts, and a note goldmark parses but this rejects is a bypass, not a
// stricter check.
//
// Erring the other way is fine — a false denial is loud and recoverable — so
// leading blank lines count here even though goldmark requires the fence on
// line 0.
func declaresRole(content string) bool {
	body := strings.ReplaceAll(content, "\r\n", "\n")
	body = strings.TrimPrefix(body, "\uFEFF")
	body = strings.TrimLeft(body, "\n")
	lines := strings.Split(body, "\n")
	if !isFrontmatterFence(lines[0]) {
		return false
	}
	var meta map[string]any
	if err := yaml.Unmarshal([]byte(frontmatterBody(lines[1:])), &meta); err != nil {
		// Unparseable frontmatter yields no meta in trip2g either, so the note
		// cannot become a role through it.
		return false
	}
	_, found := meta["fleet_id"]
	return found
}

// isFrontmatterFence mirrors goldmark-meta's isSeparator: ANY run of dashes,
// surrounded by optional whitespace. Not just "---" — "----" opens frontmatter
// just as well, and the opening and closing fences need not be the same length.
func isFrontmatterFence(line string) bool {
	trimmed := strings.TrimSpace(line)
	return trimmed != "" && strings.Trim(trimmed, "-") == ""
}

// frontmatterBody returns the lines up to the closing fence — or all of them
// when there is none. An unterminated block is NOT "not frontmatter": goldmark
// closes the open block at EOF and parses what it collected, so a note ending
// mid-frontmatter still carries meta.
func frontmatterBody(lines []string) string {
	for i, line := range lines {
		if isFrontmatterFence(line) {
			return strings.Join(lines[:i], "\n")
		}
	}
	return strings.Join(lines, "\n")
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
