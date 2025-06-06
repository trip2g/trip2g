package updatenotegraphpositions

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go . Env

func TestResolve(t *testing.T) {
	type args struct {
		ctx   context.Context
		input model.UpdateNoteGraphPositionsInput
	}
	tests := []struct {
		name          string
		env           Env
		args          args
		want          *model.UpdateNoteGraphPositionsPayload
		wantErr       bool
		afterCallback func(t *testing.T, mockEnv *EnvMock)
	}{
		{
			name: "successful positions update",
			env: &EnvMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				UpdateNoteGraphPositionByPathIDFunc: func(ctx context.Context, arg db.UpdateNoteGraphPositionByPathIDParams) error {
					return nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateNoteGraphPositionsInput{
					Positions: []model.UpdateNoteGraphPositionInput{
						{PathID: 123, X: 250.5, Y: 180.7},
						{PathID: 456, X: 100.0, Y: 200.0},
					},
				},
			},
			want: &model.UpdateNoteGraphPositionsPayload{
				Success: true,
				PathsID: []int64{123, 456},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *EnvMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Len(t, mockEnv.UpdateNoteGraphPositionByPathIDCalls(), 2)

				// Verify correct parameters were passed for first position
				updateCall1 := mockEnv.UpdateNoteGraphPositionByPathIDCalls()[0]
				require.Equal(t, int64(123), updateCall1.Arg.ID)
				require.True(t, updateCall1.Arg.GraphPositionX.Valid)
				require.Equal(t, 250.5, updateCall1.Arg.GraphPositionX.Float64)
				require.True(t, updateCall1.Arg.GraphPositionY.Valid)
				require.Equal(t, 180.7, updateCall1.Arg.GraphPositionY.Float64)

				// Verify correct parameters were passed for second position
				updateCall2 := mockEnv.UpdateNoteGraphPositionByPathIDCalls()[1]
				require.Equal(t, int64(456), updateCall2.Arg.ID)
				require.True(t, updateCall2.Arg.GraphPositionX.Valid)
				require.Equal(t, 100.0, updateCall2.Arg.GraphPositionX.Float64)
				require.True(t, updateCall2.Arg.GraphPositionY.Valid)
				require.Equal(t, 200.0, updateCall2.Arg.GraphPositionY.Float64)
			},
		},
		{
			name: "admin token error",
			env: &EnvMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return nil, errors.New("unauthorized")
				},
				UpdateNoteGraphPositionByPathIDFunc: func(ctx context.Context, arg db.UpdateNoteGraphPositionByPathIDParams) error {
					return nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateNoteGraphPositionsInput{
					Positions: []model.UpdateNoteGraphPositionInput{
						{PathID: 123, X: 250.5, Y: 180.7},
					},
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *EnvMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Empty(t, mockEnv.UpdateNoteGraphPositionByPathIDCalls())
			},
		},
		{
			name: "database update error on first position",
			env: &EnvMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				UpdateNoteGraphPositionByPathIDFunc: func(ctx context.Context, arg db.UpdateNoteGraphPositionByPathIDParams) error {
					return errors.New("database error")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateNoteGraphPositionsInput{
					Positions: []model.UpdateNoteGraphPositionInput{
						{PathID: 999, X: 100.0, Y: 200.0},
						{PathID: 888, X: 300.0, Y: 400.0},
					},
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *EnvMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Len(t, mockEnv.UpdateNoteGraphPositionByPathIDCalls(), 1)

				// Verify parameters were passed correctly for the first position that failed
				updateCall := mockEnv.UpdateNoteGraphPositionByPathIDCalls()[0]
				require.Equal(t, int64(999), updateCall.Arg.ID)
				require.Equal(t, 100.0, updateCall.Arg.GraphPositionX.Float64)
				require.Equal(t, 200.0, updateCall.Arg.GraphPositionY.Float64)
			},
		},
		{
			name: "zero coordinates",
			env: &EnvMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				UpdateNoteGraphPositionByPathIDFunc: func(ctx context.Context, arg db.UpdateNoteGraphPositionByPathIDParams) error {
					return nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateNoteGraphPositionsInput{
					Positions: []model.UpdateNoteGraphPositionInput{
						{PathID: 456, X: 0.0, Y: 0.0},
					},
				},
			},
			want: &model.UpdateNoteGraphPositionsPayload{
				Success: true,
				PathsID: []int64{456},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *EnvMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Len(t, mockEnv.UpdateNoteGraphPositionByPathIDCalls(), 1)

				// Verify zero coordinates are handled correctly
				updateCall := mockEnv.UpdateNoteGraphPositionByPathIDCalls()[0]
				require.Equal(t, int64(456), updateCall.Arg.ID)
				require.Equal(t, 0.0, updateCall.Arg.GraphPositionX.Float64)
				require.Equal(t, 0.0, updateCall.Arg.GraphPositionY.Float64)
			},
		},
		{
			name: "empty positions list",
			env: &EnvMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				UpdateNoteGraphPositionByPathIDFunc: func(ctx context.Context, arg db.UpdateNoteGraphPositionByPathIDParams) error {
					return nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateNoteGraphPositionsInput{
					Positions: []model.UpdateNoteGraphPositionInput{},
				},
			},
			want: &model.UpdateNoteGraphPositionsPayload{
				Success: true,
				PathsID: []int64{},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *EnvMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Empty(t, mockEnv.UpdateNoteGraphPositionByPathIDCalls())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve(tt.args.ctx, tt.env, tt.args.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("Resolve() = %v, want %v", got, tt.want)
					for _, desc := range pretty.Diff(got, tt.want) {
						t.Error(desc)
					}
				}
			}

			// Verify env method calls using afterCallback
			if tt.afterCallback != nil {
				tt.afterCallback(t, tt.env.(*EnvMock))
			}
		})
	}
}
