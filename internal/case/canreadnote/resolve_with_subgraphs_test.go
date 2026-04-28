package canreadnote_test

import (
	"context"
	"testing"

	"trip2g/internal/case/canreadnote"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

type resolveWithSubgraphsEnv struct{}

func TestResolveWithSubgraphs(t *testing.T) {
	tests := []struct {
		name      string
		note      *model.NoteView
		allowed   []string
		wantAllow bool
	}{
		{
			name:      "empty scope can read free note",
			note:      &model.NoteView{Free: true},
			wantAllow: true,
		},
		{
			name:      "empty scope cannot read paid general note",
			note:      &model.NoteView{Free: false},
			wantAllow: false,
		},
		{
			name:      "non-empty scope can read paid general note",
			note:      &model.NoteView{Free: false},
			allowed:   []string{"team"},
			wantAllow: true,
		},
		{
			name:      "matching subgraph allows note",
			note:      &model.NoteView{SubgraphNames: []string{"team"}},
			allowed:   []string{"team"},
			wantAllow: true,
		},
		{
			name:      "non-matching subgraph denies note",
			note:      &model.NoteView{SubgraphNames: []string{"team"}},
			allowed:   []string{"other"},
			wantAllow: false,
		},
		{
			name: "require signin subgraph allows authenticated federated caller",
			note: &model.NoteView{
				Subgraphs: map[string]*model.NoteSubgraph{
					"signin": {RequireSignin: true},
				},
			},
			allowed:   []string{"team"},
			wantAllow: true,
		},
		{
			name: "require signin subgraph denies anonymous federated caller",
			note: &model.NoteView{
				Subgraphs: map[string]*model.NoteSubgraph{
					"signin": {RequireSignin: true},
				},
			},
			wantAllow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := canreadnote.ResolveWithSubgraphs(context.Background(), resolveWithSubgraphsEnv{}, tt.note, tt.allowed)

			require.NoError(t, err)
			require.Equal(t, tt.wantAllow, got)
		})
	}
}
