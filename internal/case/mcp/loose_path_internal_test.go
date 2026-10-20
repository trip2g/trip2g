package mcp

import (
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/model"
)

func TestResolveNoteReferenceForgivesHowModelsMisspellAPath(t *testing.T) {
	wontDo := &model.NoteView{Path: "agent/what-it-wont-do.md", PathID: 1}
	tools := &model.NoteView{Path: "agent/Default-tools.md", PathID: 2}
	backupsA := &model.NoteView{Path: "devops/agent-backups.md", PathID: 3}
	backupsB := &model.NoteView{Path: "devops/agent_backups.md", PathID: 4}
	noteViews := &model.NoteViews{PathMap: map[string]*model.NoteView{
		wontDo.Path:   wontDo,
		tools.Path:    tools,
		backupsA.Path: backupsA,
		backupsB.Path: backupsB,
	}}

	tests := []struct {
		name string
		path string
		want *model.NoteView
	}{
		{name: "exact path", path: "agent/what-it-wont-do.md", want: wontDo},
		{name: "underscores for dashes", path: "agent/what_it_wont_do.md", want: wontDo},
		{name: "another case", path: "agent/default_tools.md", want: tools},
		{name: "leading slash, no extension", path: "/agent/What-It-Wont-Do", want: wontDo},
		{name: "upper-case extension", path: "AGENT/WHAT-IT-WONT-DO.MD", want: wontDo},
		{name: "spaces for dashes", path: "agent/what it wont do.md", want: wontDo},
		{name: "an exact match wins over a loose one", path: "devops/agent_backups.md", want: backupsB},
		{name: "two loose matches resolve to none", path: "devops/Agent-Backups.md", want: nil},
		{name: "no match", path: "agent/what-it-will-do.md", want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveNoteReference(noteViews, model.MCPNoteHTMLParams{Path: tt.path})
			require.Same(t, tt.want, got)

			similar := resolveSimilarReference(noteViews, model.MCPSimilarParams{Path: tt.path})
			require.Same(t, tt.want, similar)
		})
	}
}
