package banuser_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"trip2g/internal/case/admin/banuser"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg banuser_test . Env

type Env interface {
	BanUser(ctx context.Context, params db.BanUserParams) error
	ResetBanCache(ctx context.Context) error
}

type envMock = EnvMock

func TestResolve(t *testing.T) {
	type args struct {
		ctx context.Context
		req model.BanUserInput
	}

	tests := []struct {
		name          string
		env           banuser.Env
		args          args
		want          model.BanUserOrErrorPayload
		wantErr       bool
		afterCallback func(t *testing.T, mockEnv *envMock)
	}{
		{
			name: "successful user ban",
			env: &envMock{
				BanUserFunc: func(ctx context.Context, params db.BanUserParams) error {
					return nil
				},
				ResetBanCacheFunc: func(ctx context.Context) error {
					return nil
				},
			},
			args: args{
				ctx: context.Background(),
				req: model.BanUserInput{
					UserID: 123,
					Reason: "Violation of terms",
				},
			},
			want: &model.BanUserPayload{
				UserID: 123,
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.BanUserCalls()))
				require.Equal(t, 1, len(mockEnv.ResetBanCacheCalls()))

				// Verify ban parameters
				banParams := mockEnv.BanUserCalls()[0].Params
				require.Equal(t, int64(123), banParams.UserID)
				require.Equal(t, "Violation of terms", banParams.Reason)
			},
		},
		{
			name: "error - user already banned",
			env: &envMock{
				BanUserFunc: func(ctx context.Context, params db.BanUserParams) error {
					return errors.New("UNIQUE constraint failed: user_bans.user_id")
				},
			},
			args: args{
				ctx: context.Background(),
				req: model.BanUserInput{
					UserID: 123,
					Reason: "Violation of terms",
				},
			},
			want:    &model.ErrorPayload{Message: "User already banned"},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.BanUserCalls()))
				require.Equal(t, 0, len(mockEnv.ResetBanCacheCalls())) // cache reset not called on error
			},
		},
		{
			name: "error - database error during ban",
			env: &envMock{
				BanUserFunc: func(ctx context.Context, params db.BanUserParams) error {
					return errors.New("database connection failed")
				},
			},
			args: args{
				ctx: context.Background(),
				req: model.BanUserInput{
					UserID: 456,
					Reason: "Spam",
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.BanUserCalls()))
				require.Equal(t, 0, len(mockEnv.ResetBanCacheCalls()))
			},
		},
		{
			name: "error - cache reset fails",
			env: &envMock{
				BanUserFunc: func(ctx context.Context, params db.BanUserParams) error {
					return nil
				},
				ResetBanCacheFunc: func(ctx context.Context) error {
					return errors.New("cache service unavailable")
				},
			},
			args: args{
				ctx: context.Background(),
				req: model.BanUserInput{
					UserID: 789,
					Reason: "Inappropriate content",
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Equal(t, 1, len(mockEnv.BanUserCalls()))
				require.Equal(t, 1, len(mockEnv.ResetBanCacheCalls()))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := banuser.Resolve(tt.args.ctx, tt.env, tt.args.req)
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