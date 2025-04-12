package requestemailsignin

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"trip2g/internal/db"

	"github.com/kr/pretty"
)

//go:generate go run github.com/matryer/moq@latest -out mocks_test.go . Env

func TestResolve(t *testing.T) {
	type args struct {
		ctx context.Context
		env Env
		req Request
	}
	tests := []struct {
		name    string
		args    args
		want    *Response
		wantErr bool
	}{
		{
			name: "successful request",
			args: args{
				ctx: context.Background(),
				env: &EnvMock{
					GetUserByEmailFunc: func(ctx context.Context, email string) (db.User, error) {
						return db.User{ID: 1, Email: email}, nil
					},
					CountActiveSignInCodesFunc: func(ctx context.Context, userID int64) (int64, error) {
						return 0, nil
					},
					CreateSignInCodeFunc: func(ctx context.Context, userID int64) (int64, error) {
						return 123456, nil
					},
					QueueRequestSignInEmailFunc: func(ctx context.Context, email string, code int64) error {
						return nil
					},
				},
				req: Request{
					Email: "user@example.com",
				},
			},
			want: &Response{
				Success: true,
				Errors:  nil,
			},
			wantErr: false,
		},
		{
			name: "user not found",
			args: args{
				ctx: context.Background(),
				env: &EnvMock{
					GetUserByEmailFunc: func(ctx context.Context, email string) (db.User, error) {
						return db.User{}, sql.ErrNoRows
					},
				},
				req: Request{
					Email: "nonexistent@example.com",
				},
			},
			want: &Response{
				Success: false,
				Errors:  []string{"user_not_found"},
			},
			wantErr: false,
		},
		{
			name: "too many active codes",
			args: args{
				ctx: context.Background(),
				env: &EnvMock{
					GetUserByEmailFunc: func(ctx context.Context, email string) (db.User, error) {
						return db.User{ID: 1, Email: email}, nil
					},
					CountActiveSignInCodesFunc: func(ctx context.Context, userID int64) (int64, error) {
						return 4, nil
					},
				},
				req: Request{
					Email: "user@example.com",
				},
			},
			want: &Response{
				Success: false,
				Errors:  []string{"too_many_sign_in_codes"},
			},
			wantErr: false,
		},
		{
			name: "error getting user",
			args: args{
				ctx: context.Background(),
				env: &EnvMock{
					GetUserByEmailFunc: func(ctx context.Context, email string) (db.User, error) {
						return db.User{}, errors.New("database error")
					},
				},
				req: Request{
					Email: "user@example.com",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "error counting active codes",
			args: args{
				ctx: context.Background(),
				env: &EnvMock{
					GetUserByEmailFunc: func(ctx context.Context, email string) (db.User, error) {
						return db.User{ID: 1, Email: email}, nil
					},
					CountActiveSignInCodesFunc: func(ctx context.Context, userID int64) (int64, error) {
						return 0, errors.New("database error")
					},
				},
				req: Request{
					Email: "user@example.com",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "error creating sign-in code",
			args: args{
				ctx: context.Background(),
				env: &EnvMock{
					GetUserByEmailFunc: func(ctx context.Context, email string) (db.User, error) {
						return db.User{ID: 1, Email: email}, nil
					},
					CountActiveSignInCodesFunc: func(ctx context.Context, userID int64) (int64, error) {
						return 0, nil
					},
					CreateSignInCodeFunc: func(ctx context.Context, userID int64) (int64, error) {
						return 0, errors.New("database error")
					},
				},
				req: Request{
					Email: "user@example.com",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "error queueing email",
			args: args{
				ctx: context.Background(),
				env: &EnvMock{
					GetUserByEmailFunc: func(ctx context.Context, email string) (db.User, error) {
						return db.User{ID: 1, Email: email}, nil
					},
					CountActiveSignInCodesFunc: func(ctx context.Context, userID int64) (int64, error) {
						return 0, nil
					},
					CreateSignInCodeFunc: func(ctx context.Context, userID int64) (int64, error) {
						return 123456, nil
					},
					QueueRequestSignInEmailFunc: func(ctx context.Context, email string, code int64) error {
						return errors.New("email service error")
					},
				},
				req: Request{
					Email: "user@example.com",
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve(tt.args.ctx, tt.args.env, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Resolve() = %v, want %v", got, tt.want)
				for _, desc := range pretty.Diff(got, tt.want) {
					t.Error(desc)
				}
			}
		})
	}
}
