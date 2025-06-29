package createemailwaitlistrequest

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/db"
	"trip2g/internal/graph/model"
)

//go:generate moq -out mocks_test.go . Env

func TestResolve(t *testing.T) {
	tests := []struct {
		name             string
		input            Input
		setup            func(env *EnvMock)
		wantErr          bool
		wantErrorPayload bool
		want             Payload
	}{
		{
			name: "success",
			input: Input{
				Email:  "test@example.com",
				PathID: 123,
			},
			setup: func(env *EnvMock) {
				env.RequestIPFunc = func(ctx context.Context) string {
					return "192.168.1.1"
				}
				env.InsertWaitListEmailRequestFunc = func(ctx context.Context, arg db.InsertWaitListEmailRequestParams) error {
					require.Equal(t, "test@example.com", arg.Email)
					require.Equal(t, int64(123), arg.NotePathID)
					require.Equal(t, sql.NullString{String: "192.168.1.1", Valid: true}, arg.Ip)
					return nil
				}
			},
			want: &model.CreateEmailWaitListRequestPayload{Success: true},
		},
		{
			name: "invalid email",
			input: Input{
				Email:  "invalid-email",
				PathID: 123,
			},
			setup: func(env *EnvMock) {
			},
			wantErrorPayload: true,
		},
		{
			name: "missing email",
			input: Input{
				Email:  "",
				PathID: 123,
			},
			setup: func(env *EnvMock) {
			},
			wantErrorPayload: true,
		},
		{
			name: "missing path ID",
			input: Input{
				Email:  "test@example.com",
				PathID: 0,
			},
			setup: func(env *EnvMock) {
			},
			wantErrorPayload: true,
		},
		{
			name: "empty IP address",
			input: Input{
				Email:  "test@example.com",
				PathID: 123,
			},
			setup: func(env *EnvMock) {
				env.RequestIPFunc = func(ctx context.Context) string {
					return ""
				}
				env.InsertWaitListEmailRequestFunc = func(ctx context.Context, arg db.InsertWaitListEmailRequestParams) error {
					require.Equal(t, sql.NullString{String: "", Valid: false}, arg.Ip)
					return nil
				}
			},
			want: &model.CreateEmailWaitListRequestPayload{Success: true},
		},
		{
			name: "database error",
			input: Input{
				Email:  "test@example.com",
				PathID: 123,
			},
			setup: func(env *EnvMock) {
				env.RequestIPFunc = func(ctx context.Context) string {
					return "192.168.1.1"
				}
				env.InsertWaitListEmailRequestFunc = func(ctx context.Context, arg db.InsertWaitListEmailRequestParams) error {
					return sql.ErrConnDone
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{}
			if tt.setup != nil {
				tt.setup(env)
			}

			got, err := Resolve(context.Background(), env, tt.input)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
				return
			}

			if tt.wantErrorPayload {
				require.NoError(t, err)
				require.NotNil(t, got)
				// Check that it's an ErrorPayload
				errorPayload, ok := got.(*model.ErrorPayload)
				require.True(t, ok, "Expected ErrorPayload but got %T", got)
				// Validation errors populate ByFields, not Message
				require.NotEmpty(t, errorPayload.ByFields, "Expected validation errors in ByFields")
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
