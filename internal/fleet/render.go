package fleet

import (
	"bytes"
	"strings"

	"github.com/CloudyKit/jet/v6"
)

// renderCtx is the non-secret trigger-context bag exposed to a role's Jet body.
// It NEVER carries secrets, the scoped api_token, or any credential — only the
// data trip2g already ships in the delivery payload.
type renderCtx struct {
	ChangedFiles  []changeInfo
	ChangeFile    *changeInfo
	AttachedNotes []attachedNote
	Depth         int
}

// renderInstruction renders a role-note body as a Jet template against the
// trigger context. Exactly four vars are exposed — changed_files, change_file,
// attached_notes, depth — mirroring the layoutloader Jet pattern. Referencing
// any other identifier (e.g. secrets, api_token) is an execution error, so no
// credential can leak into the instruction.
func renderInstruction(body string, ctx renderCtx) (string, error) {
	loader := jet.NewInMemLoader()
	loader.Set("role", body)
	// WithSafeWriter(nil) disables HTML auto-escaping: the rendered text is fed
	// to an LLM as a prompt, not emitted as HTML, so escaping would corrupt note
	// content (e.g. & < > inside code blocks).
	set := jet.NewSet(loader, jet.WithSafeWriter(nil))

	tmpl, err := set.GetTemplate("role")
	if err != nil {
		return "", err
	}

	vars := make(jet.VarMap)
	vars.Set(forEachChangedFiles, ctx.ChangedFiles)
	vars.Set("change_file", ctx.ChangeFile)
	vars.Set(forEachAttachedNotes, ctx.AttachedNotes)
	vars.Set("depth", ctx.Depth)

	var buf bytes.Buffer
	if execErr := tmpl.Execute(&buf, vars, nil); execErr != nil {
		return "", execErr
	}
	return withAttachedNotesHint(buf.String(), ctx.AttachedNotes), nil
}

// RenderRoleInstruction is the exported entry point for offline role rendering
// (used by cmd/fleet --once). When targetPath is non-empty, a one-element
// changed_files list and change_file pointer are built from the target note,
// matching the for_each:changed_files render-context shape. The returned string
// is the rendered instruction ready to feed into agentruntime.Run.
func RenderRoleInstruction(role Role, targetPath, targetContent string) (string, error) {
	ctx := renderCtx{}
	if targetPath != "" {
		ci := changeInfo{Path: targetPath, Content: targetContent}
		ctx.ChangedFiles = []changeInfo{ci}
		ctx.ChangeFile = &ctx.ChangedFiles[0]
	}
	return renderInstruction(role.Body, ctx)
}

// withAttachedNotesHint appends a short line naming any attached notes whose
// paths the rendered instruction doesn't already mention, so the model knows
// they are available to read even when the role body never references them.
func withAttachedNotesHint(rendered string, notes []attachedNote) string {
	if len(notes) == 0 {
		return rendered
	}
	var missing []string
	for _, n := range notes {
		if n.Path != "" && !strings.Contains(rendered, n.Path) {
			missing = append(missing, n.Path)
		}
	}
	if len(missing) == 0 {
		return rendered
	}
	return rendered + "\n\nAttached notes available: " + strings.Join(missing, ", ")
}
