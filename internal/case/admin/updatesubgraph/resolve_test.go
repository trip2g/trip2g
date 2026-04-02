package updatesubgraph_test

import (
	"context"
	"errors"
	"testing"

	"trip2g/internal/case/admin/updatesubgraph"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg updatesubgraph_test . Env

type Env interface {
	UpdateAdminSubgraph(ctx context.Context, arg db.UpdateAdminSubgraphParams) (db.Subgraph, error)
	PrepareLatestNotes(ctx context.Context, partial bool) (*appmodel.NoteViews, error)
}

type envMock = EnvMock

func TestResolve(t *testing.T) {
	tests := []struct {
		name        string
		input       model.UpdateSubgraphInput
		env         updatesubgraph.Env
		wantErr     bool
		wantErrText string
		want        model.UpdateSubgraphOrErrorPayload
	}{
		{
			name: "successful update with color",
			input: model.UpdateSubgraphInput{
				ID:            123,
				Color:         "#ff0000",
				RequireSignin: true,
			},
			env: &envMock{
				PrepareLatestNotesFunc: func(ctx context.Context, partial bool) (*appmodel.NoteViews, error) { return nil, nil },
				UpdateAdminSubgraphFunc: func(ctx context.Context, arg db.UpdateAdminSubgraphParams) (db.Subgraph, error) {
					return db.Subgraph{
						ID:            123,
						Color:         stringPtr("#ff0000"),
						RequireSignin: true,
					}, nil
				},
			},
			want: &model.UpdateSubgraphPayload{
				Subgraph: &db.Subgraph{
					ID:            123,
					Color:         stringPtr("#ff0000"),
					RequireSignin: true,
				},
			},
		},
		{
			name: "successful update without color",
			input: model.UpdateSubgraphInput{
				ID:    456,
				Color: "",
			},
			env: &envMock{
				PrepareLatestNotesFunc: func(ctx context.Context, partial bool) (*appmodel.NoteViews, error) { return nil, nil },
				UpdateAdminSubgraphFunc: func(ctx context.Context, arg db.UpdateAdminSubgraphParams) (db.Subgraph, error) {
					return db.Subgraph{
						ID: 456,
					}, nil
				},
			},
			want: &model.UpdateSubgraphPayload{
				Subgraph: &db.Subgraph{
					ID: 456,
				},
			},
		},
		{
			name: "database error",
			input: model.UpdateSubgraphInput{
				ID:    789,
				Color: "#00ff00",
			},
			env: &envMock{
				PrepareLatestNotesFunc: func(ctx context.Context, partial bool) (*appmodel.NoteViews, error) { return nil, nil },
				UpdateAdminSubgraphFunc: func(ctx context.Context, arg db.UpdateAdminSubgraphParams) (db.Subgraph, error) {
					return db.Subgraph{}, errors.New("database error")
				},
			},
			wantErr:     true,
			wantErrText: "failed to update subgraph",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := updatesubgraph.Resolve(context.Background(), tt.env, tt.input)

			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErrText)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)

			if env, ok := tt.env.(*envMock); ok {
				require.Len(t, env.UpdateAdminSubgraphCalls(), 1)
				call := env.UpdateAdminSubgraphCalls()[0]
				require.Equal(t, tt.input.ID, call.Arg.ID)

				if tt.input.Color != "" {
					require.NotNil(t, call.Arg.Color)
					require.Equal(t, tt.input.Color, *call.Arg.Color)
				} else {
					require.Nil(t, call.Arg.Color)
				}
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
