package signinbyemail

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
			name: "successful sign in",
			args: args{
				ctx: context.Background(),
				env: &EnvMock{
					VerifySignInCodeFunc: func(ctx context.Context, arg db.VerifySignInCodeParams) (int64, error) {
						return 1, nil
					},
					DeleteSignInCodesByUserIDFunc: func(ctx context.Context, userID int64) error {
						return nil
					},
					SetupUserTokenFunc: func(ctx context.Context, userID int64) (string, error) {
						return "valid_token", nil
					},
				},
				req: Request{
					Email: "user@example.com",
					Code:  123456,
				},
			},
			want: &Response{
				Token:  "valid_token",
				Errors: nil,
			},
			wantErr: false,
		},
		{
			name: "invalid code",
			args: args{
				ctx: context.Background(),
				env: &EnvMock{
					VerifySignInCodeFunc: func(ctx context.Context, arg db.VerifySignInCodeParams) (int64, error) {
						return 0, sql.ErrNoRows
					},
				},
				req: Request{
					Email: "user@example.com",
					Code:  123456,
				},
			},
			want: &Response{
				Token:  "",
				Errors: []string{"invalid_code"},
			},
			wantErr: false,
		},
		{
			name: "error verifying code",
			args: args{
				ctx: context.Background(),
				env: &EnvMock{
					VerifySignInCodeFunc: func(ctx context.Context, arg db.VerifySignInCodeParams) (int64, error) {
						return 0, errors.New("database error")
					},
				},
				req: Request{
					Email: "user@example.com",
					Code:  123456,
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "error setting up token",
			args: args{
				ctx: context.Background(),
				env: &EnvMock{
					VerifySignInCodeFunc: func(ctx context.Context, arg db.VerifySignInCodeParams) (int64, error) {
						return 1, nil
					},
					SetupUserTokenFunc: func(ctx context.Context, userID int64) (string, error) {
						return "", errors.New("token error")
					},
				},
				req: Request{
					Email: "user@example.com",
					Code:  123456,
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "error deleting codes",
			args: args{
				ctx: context.Background(),
				env: &EnvMock{
					VerifySignInCodeFunc: func(ctx context.Context, arg db.VerifySignInCodeParams) (int64, error) {
						return 1, nil
					},
					SetupUserTokenFunc: func(ctx context.Context, userID int64) (string, error) {
						return "valid_token", nil
					},
					DeleteSignInCodesByUserIDFunc: func(ctx context.Context, userID int64) error {
						return errors.New("delete error")
					},
				},
				req: Request{
					Email: "user@example.com",
					Code:  123456,
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
