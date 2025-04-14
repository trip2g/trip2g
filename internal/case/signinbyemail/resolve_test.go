package signinbyemail_test

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"trip2g/internal/case/signinbyemail"
	"trip2g/internal/db"

	"github.com/kr/pretty"
)

//go:generate go run github.com/matryer/moq@latest -out mocks_test.go . Env

type envMock = signinbyemail.EnvMock
type request = signinbyemail.Request

func TestResolve(t *testing.T) {
	type args struct {
		ctx context.Context
		env signinbyemail.Env
		req signinbyemail.Request
	}
	tests := []struct {
		name    string
		args    args
		want    *signinbyemail.Response
		wantErr bool
	}{
		{
			name: "successful sign in",
			args: args{
				ctx: context.Background(),
				env: &envMock{
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
				req: request{
					Email: "user@example.com",
					Code:  123456,
				},
			},
			want: &signinbyemail.Response{
				Token:  "valid_token",
				Errors: nil,
			},
			wantErr: false,
		},
		{
			name: "invalid code",
			args: args{
				ctx: context.Background(),
				env: &envMock{
					VerifySignInCodeFunc: func(ctx context.Context, arg db.VerifySignInCodeParams) (int64, error) {
						return 0, sql.ErrNoRows
					},
				},
				req: request{
					Email: "user@example.com",
					Code:  123456,
				},
			},
			want: &signinbyemail.Response{
				Token:  "",
				Errors: []string{"invalid_code"},
			},
			wantErr: false,
		},
		{
			name: "error verifying code",
			args: args{
				ctx: context.Background(),
				env: &envMock{
					VerifySignInCodeFunc: func(ctx context.Context, arg db.VerifySignInCodeParams) (int64, error) {
						return 0, errors.New("database error")
					},
				},
				req: request{
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
				env: &envMock{
					VerifySignInCodeFunc: func(ctx context.Context, arg db.VerifySignInCodeParams) (int64, error) {
						return 1, nil
					},
					SetupUserTokenFunc: func(ctx context.Context, userID int64) (string, error) {
						return "", errors.New("token error")
					},
				},
				req: request{
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
				env: &envMock{
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
				req: request{
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
			got, err := signinbyemail.Resolve(tt.args.ctx, tt.args.env, tt.args.req)
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
