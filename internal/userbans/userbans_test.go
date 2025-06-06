package userbans_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"trip2g/internal/db"
	"trip2g/internal/userbans"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg userbans_test . Env

type Env interface {
	ListAllUserBans(ctx context.Context) ([]db.UserBan, error)
}

type envMock = EnvMock

func TestUserBans_UserBanByUserID(t *testing.T) {
	type args struct {
		ctx    context.Context
		userID int64
	}
	tests := []struct {
		name    string
		env     userbans.Env
		args    args
		want    *db.UserBan
		wantErr bool
	}{
		{
			name: "user is banned",
			env: &envMock{
				ListAllUserBansFunc: func(ctx context.Context) ([]db.UserBan, error) {
					return []db.UserBan{
						{UserID: 1, Reason: "Spam"},
						{UserID: 2, Reason: "Harassment"},
					}, nil
				},
			},
			args: args{
				ctx:    context.Background(),
				userID: 1,
			},
			want: &db.UserBan{
				UserID: 1,
				Reason: "Spam",
			},
			wantErr: false,
		},
		{
			name: "user is not banned",
			env: &envMock{
				ListAllUserBansFunc: func(ctx context.Context) ([]db.UserBan, error) {
					return []db.UserBan{
						{UserID: 1, Reason: "Spam"},
						{UserID: 2, Reason: "Harassment"},
					}, nil
				},
			},
			args: args{
				ctx:    context.Background(),
				userID: 99,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "database error",
			env: &envMock{
				ListAllUserBansFunc: func(ctx context.Context) ([]db.UserBan, error) {
					return nil, errors.New("database connection failed")
				},
			},
			args: args{
				ctx:    context.Background(),
				userID: 1,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "empty ban list",
			env: &envMock{
				ListAllUserBansFunc: func(ctx context.Context) ([]db.UserBan, error) {
					return []db.UserBan{}, nil
				},
			},
			args: args{
				ctx:    context.Background(),
				userID: 1,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "cache hit - second lookup",
			env: &envMock{
				ListAllUserBansFunc: func(ctx context.Context) ([]db.UserBan, error) {
					return []db.UserBan{
						{UserID: 1, Reason: "Spam"},
					}, nil
				},
			},
			args: args{
				ctx:    context.Background(),
				userID: 1,
			},
			want: &db.UserBan{
				UserID: 1,
				Reason: "Spam",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userBans := userbans.New(tt.env)

			// For cache hit test, do an initial lookup
			if tt.name == "cache hit - second lookup" {
				_, err := userBans.UserBanByUserID(tt.args.ctx, tt.args.userID)
				require.NoError(t, err)
			}

			got, err := userBans.UserBanByUserID(tt.args.ctx, tt.args.userID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UserBanByUserID() = %v, want %v", got, tt.want)
				for _, desc := range pretty.Diff(got, tt.want) {
					t.Error(desc)
				}
			}

			// Verify cache behavior for non-error cases
			if !tt.wantErr {
				mockEnv := tt.env.(*envMock)
				if tt.name == "cache hit - second lookup" {
					// Should have been called only once (during first lookup)
					require.Len(t, mockEnv.ListAllUserBansCalls(), 1)
				} else {
					// Should have been called once
					require.Len(t, mockEnv.ListAllUserBansCalls(), 1)
				}
			}
		})
	}
}

func TestUserBans_ResetBanCache(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name       string
		env        userbans.Env
		args       args
		setupFunc  func(*userbans.UserBans, context.Context)
		wantErr    bool
		checkCalls func(*testing.T, *envMock)
	}{
		{
			name: "reset empty cache",
			env: &envMock{
				ListAllUserBansFunc: func(ctx context.Context) ([]db.UserBan, error) {
					return []db.UserBan{{UserID: 1, Reason: "Test"}}, nil
				},
			},
			args: args{
				ctx: context.Background(),
			},
			setupFunc: func(ub *userbans.UserBans, ctx context.Context) {
				// No setup - cache is empty
			},
			wantErr: false,
			checkCalls: func(t *testing.T, mock *envMock) {
				// Should be called once during reset
				require.Len(t, mock.ListAllUserBansCalls(), 1)
			},
		},
		{
			name: "reset loaded cache",
			env: &envMock{
				ListAllUserBansFunc: func(ctx context.Context) ([]db.UserBan, error) {
					return []db.UserBan{{UserID: 1, Reason: "Test"}}, nil
				},
			},
			args: args{
				ctx: context.Background(),
			},
			setupFunc: func(ub *userbans.UserBans, ctx context.Context) {
				// Load cache first
				_, err := ub.UserBanByUserID(ctx, 1)
				require.NoError(t, err)
			},
			wantErr: false,
			checkCalls: func(t *testing.T, mock *envMock) {
				// Should be called twice: once during setup, once during reset
				require.Len(t, mock.ListAllUserBansCalls(), 2)
			},
		},
		{
			name: "database error during reset",
			env: &envMock{
				ListAllUserBansFunc: func(ctx context.Context) ([]db.UserBan, error) {
					return nil, errors.New("database error")
				},
			},
			args: args{
				ctx: context.Background(),
			},
			setupFunc: func(ub *userbans.UserBans, ctx context.Context) {
				// No setup - just test error during reset
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userBans := userbans.New(tt.env)
			mockEnv := tt.env.(*envMock)

			// Setup
			if tt.setupFunc != nil {
				tt.setupFunc(userBans, tt.args.ctx)
			}

			err := userBans.ResetBanCache(tt.args.ctx)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// Verify behavior
			if tt.checkCalls != nil {
				tt.checkCalls(t, mockEnv)
			}
		})
	}
}
