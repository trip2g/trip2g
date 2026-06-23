package signinbyemail

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"

	"trip2g/internal/db"
	gmodel "trip2g/internal/graph/model"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name    string
		env     Env
		req     gmodel.SignInByEmailInput
		want    gmodel.SignInOrErrorPayload
		wantErr bool
	}{
		{
			name: "invalid code - sql.ErrNoRows returns code field error",
			env: &EnvMock{
				VerifySignInCodeFunc: func(ctx context.Context, arg db.VerifySignInCodeParams) (int64, error) {
					return 0, sql.ErrNoRows
				},
			},
			req: gmodel.SignInByEmailInput{
				Email: "user@example.com",
				Code:  "123456",
			},
			want: &gmodel.ErrorPayload{
				ByFields: []gmodel.FieldMessage{
					{Name: "code", Value: "Code is invalid or expired"},
				},
			},
			wantErr: false,
		},
		{
			name: "system error propagated",
			env: &EnvMock{
				VerifySignInCodeFunc: func(ctx context.Context, arg db.VerifySignInCodeParams) (int64, error) {
					return 0, errors.New("db connection lost")
				},
			},
			req: gmodel.SignInByEmailInput{
				Email: "user@example.com",
				Code:  "123456",
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve(context.Background(), tt.env, tt.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Resolve() mismatch")
				for _, desc := range pretty.Diff(got, tt.want) {
					t.Error(desc)
				}
			}
		})
	}
}
