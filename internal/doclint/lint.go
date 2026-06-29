package doclint

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"
	"trip2g/internal/noteloader"
)

// lintLine is a single lint finding collected before printing.
type lintLine struct {
	path    string
	message string
	level   model.NoteWarningLevel
}

// Run runs the DB-free linter over dir, writes sorted results to w and returns
// exit code 1 if any warnings were found, 0 if clean.
// log may be nil (falls back to DummyLogger).
func Run(ctx context.Context, dir string, w io.Writer, log logger.Logger) (exitCode int, err error) {
	if log == nil {
		log = &logger.DummyLogger{}
	}

	env := newFsEnv(dir, log)
	ldr := noteloader.New("lint", env, mdloader.Config{})

	if loadErr := ldr.Load(ctx, noteloader.LoadOptions{SkipSearchIndex: true}); loadErr != nil {
		return 1, fmt.Errorf("doclint: load failed: %w", loadErr)
	}

	nvs := ldr.NoteViews()
	layouts := ldr.Layouts()

	var lines []lintLine

	// --- per-note warnings from the loader pipeline ---
	for _, note := range nvs.List {
		for _, nw := range note.Warnings {
			lines = append(lines, lintLine{
				path:    note.Path,
				message: nw.Message,
				level:   nw.Level,
			})
		}

		// --- lint-added checks on ResolvedLinks ---
		noteLang := pathLangPrefix(note.Path)
		for target, permalink := range note.ResolvedLinks {
			// Only bare wikilinks (no path separator) are checked.
			if strings.Contains(target, "/") {
				continue
			}

			// Ambiguous bare wikilink: basename maps to >1 note.
			basename := strings.ToLower(target)
			if candidates := nvs.BasenameMap[basename]; len(candidates) > 1 {
				paths := make([]string, len(candidates))
				for i, c := range candidates {
					paths[i] = c.Path
				}
				lines = append(lines, lintLine{
					path: note.Path,
					message: fmt.Sprintf(
						"ambiguous bare wikilink [[%s]]: %d candidates (%s)",
						target, len(candidates), strings.Join(paths, ", "),
					),
					level: model.NoteWarningWarning,
				})
			}

			// Cross-language leak: bare wikilink in an en/ note resolves
			// under ru/ (or vice-versa), keyed on path prefix not frontmatter lang.
			if noteLang != "" {
				other := otherLang(noteLang)
				if strings.HasPrefix(permalink, "/"+other+"/") || permalink == "/"+other {
					lines = append(lines, lintLine{
						path: note.Path,
						message: fmt.Sprintf(
							"cross-language wikilink leak: [[%s]] resolves to %s (note lang=%s, link lang=%s)",
							target, permalink, noteLang, other,
						),
						level: model.NoteWarningWarning,
					})
				}
			}
		}
	}

	// --- layout render warnings (from smokeRenderLayouts) ---
	if layouts != nil {
		for _, layout := range layouts.Map {
			for _, lw := range layout.Warnings {
				lines = append(lines, lintLine{
					path:    layout.Path,
					message: lw.Message,
					level:   lw.Level,
				})
			}
		}
	}

	// Sort by path, then message for deterministic output.
	sort.Slice(lines, func(i, j int) bool {
		if lines[i].path != lines[j].path {
			return lines[i].path < lines[j].path
		}
		return lines[i].message < lines[j].message
	})

	for _, line := range lines {
		fmt.Fprintf(w, "%s:0: %s %s\n", line.path, levelStr(line.level), line.message)
	}

	if len(lines) > 0 {
		return 1, nil
	}
	return 0, nil
}

// pathLangPrefix returns "en" or "ru" if the relative note path starts with
// that prefix, otherwise "".
func pathLangPrefix(path string) string {
	if strings.HasPrefix(path, "en/") {
		return "en"
	}
	if strings.HasPrefix(path, "ru/") {
		return "ru"
	}
	return ""
}

// otherLang returns the opposite language code for en/ru pairs.
func otherLang(lang string) string {
	switch lang {
	case "en":
		return "ru"
	case "ru":
		return "en"
	}
	return ""
}

// levelStr converts a NoteWarningLevel to its string label.
func levelStr(l model.NoteWarningLevel) string {
	switch l {
	case model.NoteWarningWarning:
		return "warning"
	case model.NoteWarningCritical:
		return "critical"
	default:
		return "info"
	}
}
