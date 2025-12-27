package deleteadmin_test

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"

	"trip2g/internal/case/admin/deleteadmin"
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
		input model.DeleteAdminInput
	}

	tests := []struct {
		name          string
		env           deleteadmin.Env
		args          args
		want          model.DeleteAdminOrErrorPayload
		wantErr       bool
		afterCallback func(t *testing.T, mockEnv *envMock)
	}{
		{
			name: "successful admin deletion",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				},
				AdminByUserIDFunc: func(ctx context.Context, userID int64) (db.Admin, error) {
					return db.Admin{UserID: 2}, nil
				},
				DeleteAdminFunc: func(ctx context.Context, userID int64) error {
					return nil
				},
			},
			args: args{
				ctx:   context.Background(),
				input: model.DeleteAdminInput{UserID: 2},
			},
			want:    &model.DeleteAdminPayload{Success: true},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.DeleteAdminCalls(), 1)
				require.Equal(t, int64(2), mockEnv.DeleteAdminCalls()[0].UserID)
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
				input: model.DeleteAdminInput{UserID: 2},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "error - cannot delete self",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				},
			},
			args: args{
				ctx:   context.Background(),
				input: model.DeleteAdminInput{UserID: 1},
			},
			want:    &model.ErrorPayload{Message: "Cannot delete yourself as admin"},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Empty(t, mockEnv.DeleteAdminCalls())
			},
		},
		{
			name: "error - user is not admin",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				},
				AdminByUserIDFunc: func(ctx context.Context, userID int64) (db.Admin, error) {
					return db.Admin{}, sql.ErrNoRows
				},
			},
			args: args{
				ctx:   context.Background(),
				input: model.DeleteAdminInput{UserID: 2},
			},
			want:    &model.ErrorPayload{Message: "User is not an admin"},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Empty(t, mockEnv.DeleteAdminCalls())
			},
		},
		{
			name: "error - database error during delete",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1, Role: "admin"}, nil
				},
				AdminByUserIDFunc: func(ctx context.Context, userID int64) (db.Admin, error) {
					return db.Admin{UserID: 2}, nil
				},
				DeleteAdminFunc: func(ctx context.Context, userID int64) error {
					return errors.New("database error")
				},
			},
			args: args{
				ctx:   context.Background(),
				input: model.DeleteAdminInput{UserID: 2},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := deleteadmin.Resolve(tt.args.ctx, tt.env, tt.args.input)
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
