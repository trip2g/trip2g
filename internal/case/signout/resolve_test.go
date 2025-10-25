package signout_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"trip2g/internal/case/signout"
	gmodel "trip2g/internal/graph/model"
	"trip2g/internal/model"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)


type envMock = EnvMock

func TestResolve(t *testing.T) {
	type args struct {
		ctx context.Context
	}

	tests := []struct {
		name          string
		env           signout.Env
		args          args
		want          gmodel.SignOutOrErrorPayload
		wantErr       bool
		afterCallback func(t *testing.T, mockEnv *envMock)
	}{
		{
			name: "successful sign out",
			env: &envMock{
				ResetUserTokenFunc: func(ctx context.Context) error {
					return nil
				},
			},
			args: args{
				ctx: context.Background(),
			},
			want: &gmodel.SignOutPayload{
				Viewer: &model.Viewer{},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.ResetUserTokenCalls(), 1)
			},
		},
		{
			name: "error - reset token fails",
			env: &envMock{
				ResetUserTokenFunc: func(ctx context.Context) error {
					return errors.New("token reset failed")
				},
			},
			args: args{
				ctx: context.Background(),
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.ResetUserTokenCalls(), 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := signout.Resolve(tt.args.ctx, tt.env)
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

			if tt.afterCallback != nil {
				mockEnv := tt.env.(*envMock)
				tt.afterCallback(t, mockEnv)
			}
		})
	}
}
