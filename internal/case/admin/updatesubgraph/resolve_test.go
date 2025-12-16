package updatesubgraph_test

import (
	"context"
	"errors"
	"testing"

	"trip2g/internal/case/admin/updatesubgraph"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"

	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg updatesubgraph_test . Env

type Env interface {
	UpdateAdminSubgraph(ctx context.Context, arg db.UpdateAdminSubgraphParams) (db.Subgraph, error)
}

type envMock = EnvMock

func TestRequest_Resolve(t *testing.T) {
	type fields struct {
		ID    int64
		Color string
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name        string
		fields      fields
		env         updatesubgraph.Env
		args        args
		want        model.UpdateSubgraphOrErrorPayload
		wantErr     bool
		wantErrText string
	}{
		{
			name: "successful update with color",
			fields: fields{
				ID:    123,
				Color: "#ff0000",
			},
			env: &envMock{
				UpdateAdminSubgraphFunc: func(ctx context.Context, arg db.UpdateAdminSubgraphParams) (db.Subgraph, error) {
					return db.Subgraph{
						ID:    123,
						Color: stringPtr("#ff0000"),
					}, nil
				},
			},
			args: args{
				ctx: context.Background(),
			},
			want: &model.UpdateSubgraphPayload{
				Subgraph: &db.Subgraph{
					ID:    123,
					Color: stringPtr("#ff0000"),
				},
			},
		},
		{
			name: "successful update without color",
			fields: fields{
				ID:    456,
				Color: "",
			},
			env: &envMock{
				UpdateAdminSubgraphFunc: func(ctx context.Context, arg db.UpdateAdminSubgraphParams) (db.Subgraph, error) {
					return db.Subgraph{
						ID: 456,
					}, nil
				},
			},
			args: args{
				ctx: context.Background(),
			},
			want: &model.UpdateSubgraphPayload{
				Subgraph: &db.Subgraph{
					ID: 456,
				},
			},
		},
		{
			name: "database error",
			fields: fields{
				ID:    789,
				Color: "#00ff00",
			},
			env: &envMock{
				UpdateAdminSubgraphFunc: func(ctx context.Context, arg db.UpdateAdminSubgraphParams) (db.Subgraph, error) {
					return db.Subgraph{}, errors.New("database error")
				},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr:     true,
			wantErrText: "failed to update subgraph",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &updatesubgraph.Request{
				ID:    tt.fields.ID,
				Color: tt.fields.Color,
			}
			got, err := req.Resolve(tt.args.ctx, tt.env)

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
				require.Equal(t, tt.fields.ID, call.Arg.ID)

				if tt.fields.Color != "" {
					require.NotNil(t, call.Arg.Color)
					require.Equal(t, tt.fields.Color, *call.Arg.Color)
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
