package unbanuser_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"trip2g/internal/case/admin/unbanuser"
	"trip2g/internal/graph/model"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg unbanuser_test . Env

type Env interface {
	UnbanUser(ctx context.Context, userID int64) error
	ResetBanCache(ctx context.Context) error
}

type envMock = EnvMock

func TestResolve(t *testing.T) {
	type args struct {
		ctx context.Context
		req model.UnbanUserInput
	}

	tests := []struct {
		name          string
		env           unbanuser.Env
		args          args
		want          model.UnbanUserOrErrorPayload
		wantErr       bool
		afterCallback func(t *testing.T, mockEnv *envMock)
	}{
		{
			name: "successful user unban",
			env: &envMock{
				UnbanUserFunc: func(ctx context.Context, userID int64) error {
					return nil
				},
				ResetBanCacheFunc: func(ctx context.Context) error {
					return nil
				},
			},
			args: args{
				ctx: context.Background(),
				req: model.UnbanUserInput{
					UserID: 123,
				},
			},
			want: &model.UnbanUserPayload{
				UserID: 123,
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.UnbanUserCalls()))
				require.Equal(t, 1, len(mockEnv.ResetBanCacheCalls()))

				// Verify unban parameters
				require.Equal(t, int64(123), mockEnv.UnbanUserCalls()[0].UserID)
			},
		},
		{
			name: "error - database error during unban",
			env: &envMock{
				UnbanUserFunc: func(ctx context.Context, userID int64) error {
					return errors.New("database connection failed")
				},
			},
			args: args{
				ctx: context.Background(),
				req: model.UnbanUserInput{
					UserID: 456,
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.UnbanUserCalls()))
				require.Equal(t, 0, len(mockEnv.ResetBanCacheCalls())) // cache reset not called on error
			},
		},
		{
			name: "error - cache reset fails",
			env: &envMock{
				UnbanUserFunc: func(ctx context.Context, userID int64) error {
					return nil
				},
				ResetBanCacheFunc: func(ctx context.Context) error {
					return errors.New("cache service unavailable")
				},
			},
			args: args{
				ctx: context.Background(),
				req: model.UnbanUserInput{
					UserID: 789,
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.UnbanUserCalls()))
				require.Equal(t, 1, len(mockEnv.ResetBanCacheCalls()))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := unbanuser.Resolve(tt.args.ctx, tt.env, tt.args.req)
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