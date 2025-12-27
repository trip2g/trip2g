package createadmin_test

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	"trip2g/internal/case/admin/createadmin"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

type envMock = EnvMock

func TestResolve(t *testing.T) {
	type args struct {
		ctx   context.Context
		input model.CreateAdminInput
	}

	tests := []struct {
		name          string
		env           createadmin.Env
		args          args
		want          model.CreateAdminOrErrorPayload
		wantErr       bool
		afterCallback func(t *testing.T, mockEnv *envMock)
	}{
		{
			name: "successful admin creation",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				},
				UserByIDFunc: func(ctx context.Context, id int64) (db.User, error) {
					return db.User{ID: 2}, nil
				},
				AdminByUserIDFunc: func(ctx context.Context, userID int64) (db.Admin, error) {
					return db.Admin{}, sql.ErrNoRows
				},
				InsertAdminFunc: func(ctx context.Context, arg db.InsertAdminParams) (db.Admin, error) {
					grantedBy := int64(1)
					return db.Admin{
						UserID:    arg.UserID,
						GrantedAt: time.Now(),
						GrantedBy: &grantedBy,
					}, nil
				},
			},
			args: args{
				ctx:   context.Background(),
				input: model.CreateAdminInput{UserID: 2},
			},
			want: &model.CreateAdminPayload{
				Admin: &db.Admin{
					UserID:    2,
					GrantedAt: time.Now(),
					GrantedBy: func() *int64 { v := int64(1); return &v }(),
				},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.InsertAdminCalls(), 1)
				require.Equal(t, int64(2), mockEnv.InsertAdminCalls()[0].Arg.UserID)
				require.Equal(t, int64(1), *mockEnv.InsertAdminCalls()[0].Arg.GrantedBy)
			},
		},
		{
			name: "error - not authenticated",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return nil, errors.New("not authenticated")
				},
			},
			args: args{
				ctx:   context.Background(),
				input: model.CreateAdminInput{UserID: 2},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "error - user not found",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				},
				UserByIDFunc: func(ctx context.Context, id int64) (db.User, error) {
					return db.User{}, sql.ErrNoRows
				},
			},
			args: args{
				ctx:   context.Background(),
				input: model.CreateAdminInput{UserID: 999},
			},
			want:    &model.ErrorPayload{Message: "User not found"},
			wantErr: false,
		},
		{
			name: "error - user already admin",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				},
				UserByIDFunc: func(ctx context.Context, id int64) (db.User, error) {
					return db.User{ID: 2}, nil
				},
				AdminByUserIDFunc: func(ctx context.Context, userID int64) (db.Admin, error) {
					return db.Admin{UserID: 2}, nil
				},
			},
			args: args{
				ctx:   context.Background(),
				input: model.CreateAdminInput{UserID: 2},
			},
			want:    &model.ErrorPayload{Message: "User is already an admin"},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Empty(t, mockEnv.InsertAdminCalls())
			},
		},
		{
			name: "error - database error during insert",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				},
				UserByIDFunc: func(ctx context.Context, id int64) (db.User, error) {
					return db.User{ID: 2}, nil
				},
				AdminByUserIDFunc: func(ctx context.Context, userID int64) (db.Admin, error) {
					return db.Admin{}, sql.ErrNoRows
				},
				InsertAdminFunc: func(ctx context.Context, arg db.InsertAdminParams) (db.Admin, error) {
					return db.Admin{}, errors.New("database error")
				},
			},
			args: args{
				ctx:   context.Background(),
				input: model.CreateAdminInput{UserID: 2},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createadmin.Resolve(tt.args.ctx, tt.env, tt.args.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			// For success case, we need to compare without GrantedAt since it's time-based
			payload, isPayload := got.(*model.CreateAdminPayload)
			wantPayload, isWantPayload := tt.want.(*model.CreateAdminPayload)
			if isPayload && isWantPayload {
				require.Equal(t, wantPayload.Admin.UserID, payload.Admin.UserID)
				require.Equal(t, *wantPayload.Admin.GrantedBy, *payload.Admin.GrantedBy)
			} else if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Resolve() = %v, want %v", got, tt.want)
				for _, desc := range pretty.Diff(got, tt.want) {
					t.Error(desc)
				}
			}

			if tt.afterCallback != nil {
				mockEnv := tt.env.(*envMock)
				tt.afterCallback(t, mockEnv)
			}
		})
	}
}
