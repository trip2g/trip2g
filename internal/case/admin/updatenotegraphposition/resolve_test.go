package updatenotegraphposition_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"trip2g/internal/case/admin/updatenotegraphposition"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg updatenotegraphposition_test . Env

type Env interface {
	UpdateNoteGraphPositionByPathID(ctx context.Context, arg db.UpdateNoteGraphPositionByPathIDParams) error
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
}

type envMock = EnvMock

func TestResolve(t *testing.T) {
	type args struct {
		ctx   context.Context
		input model.UpdateNoteGraphPositionInput
	}
	tests := []struct {
		name          string
		env           updatenotegraphposition.Env
		args          args
		want          model.UpdateNoteGraphPositionOrErrorPayload
		wantErr       bool
		afterCallback func(t *testing.T, mockEnv *envMock)
	}{
		{
			name: "successful position update",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				UpdateNoteGraphPositionByPathIDFunc: func(ctx context.Context, arg db.UpdateNoteGraphPositionByPathIDParams) error {
					return nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateNoteGraphPositionInput{
					PathID: 123,
					X:      250.5,
					Y:      180.7,
				},
			},
			want: &model.UpdateNoteGraphPositionPayload{
				PathID: 123,
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.UpdateNoteGraphPositionByPathIDCalls()))

				// Verify correct parameters were passed
				updateCall := mockEnv.UpdateNoteGraphPositionByPathIDCalls()[0]
				require.Equal(t, int64(123), updateCall.Arg.ID)
				require.True(t, updateCall.Arg.GraphPositionX.Valid)
				require.Equal(t, 250.5, updateCall.Arg.GraphPositionX.Float64)
				require.True(t, updateCall.Arg.GraphPositionY.Valid)
				require.Equal(t, 180.7, updateCall.Arg.GraphPositionY.Float64)
			},
		},
		{
			name: "admin token error",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return nil, errors.New("unauthorized")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateNoteGraphPositionInput{
					PathID: 123,
					X:      250.5,
					Y:      180.7,
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 0, len(mockEnv.UpdateNoteGraphPositionByPathIDCalls()))
			},
		},
		{
			name: "database update error",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				UpdateNoteGraphPositionByPathIDFunc: func(ctx context.Context, arg db.UpdateNoteGraphPositionByPathIDParams) error {
					return errors.New("database error")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateNoteGraphPositionInput{
					PathID: 999,
					X:      100.0,
					Y:      200.0,
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.UpdateNoteGraphPositionByPathIDCalls()))

				// Verify parameters were still passed correctly
				updateCall := mockEnv.UpdateNoteGraphPositionByPathIDCalls()[0]
				require.Equal(t, int64(999), updateCall.Arg.ID)
				require.Equal(t, 100.0, updateCall.Arg.GraphPositionX.Float64)
				require.Equal(t, 200.0, updateCall.Arg.GraphPositionY.Float64)
			},
		},
		{
			name: "zero coordinates",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				UpdateNoteGraphPositionByPathIDFunc: func(ctx context.Context, arg db.UpdateNoteGraphPositionByPathIDParams) error {
					return nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UpdateNoteGraphPositionInput{
					PathID: 456,
					X:      0.0,
					Y:      0.0,
				},
			},
			want: &model.UpdateNoteGraphPositionPayload{
				PathID: 456,
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.CurrentAdminUserTokenCalls()))
				require.Equal(t, 1, len(mockEnv.UpdateNoteGraphPositionByPathIDCalls()))

				// Verify zero coordinates are handled correctly
				updateCall := mockEnv.UpdateNoteGraphPositionByPathIDCalls()[0]
				require.Equal(t, int64(456), updateCall.Arg.ID)
				require.Equal(t, 0.0, updateCall.Arg.GraphPositionX.Float64)
				require.Equal(t, 0.0, updateCall.Arg.GraphPositionY.Float64)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := updatenotegraphposition.Resolve(tt.args.ctx, tt.env, tt.args.input)
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
				mockEnv := tt.env.(*envMock)
				tt.afterCallback(t, mockEnv)
			}
		})
	}
}