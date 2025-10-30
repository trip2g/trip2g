package vacuumdatabase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trip2g/internal/case/cronjob/vacuumdatabase"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg vacuumdatabase_test . Env

type Env interface {
	VacuumDB(ctx context.Context) error
	Now() time.Time
}

func TestResolve(t *testing.T) {
	tests := []struct {
		name     string
		setupEnv func() *EnvMock
		wantErr  bool
	}{
		{
			name: "successful vacuum",
			setupEnv: func() *EnvMock {
				startTime := time.Now()
				return &EnvMock{
					VacuumDBFunc: func(ctx context.Context) error {
						// Simulate some processing time
						time.Sleep(10 * time.Millisecond)
						return nil
					},
					NowFunc: func() time.Time {
						// First call returns start time, second call returns end time
						defer func() {
							startTime = startTime.Add(15 * time.Millisecond)
						}()
						return startTime
					},
				}
			},
			wantErr: false,
		},
		{
			name: "vacuum fails",
			setupEnv: func() *EnvMock {
				return &EnvMock{
					VacuumDBFunc: func(ctx context.Context) error {
						return errors.New("database locked")
					},
					NowFunc: time.Now,
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := tt.setupEnv()
			ctx := context.Background()

			result, err := vacuumdatabase.Resolve(ctx, env, vacuumdatabase.Filter{})

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, result)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			require.True(t, result.Success)
			require.Greater(t, result.Duration, time.Duration(0))
		})
	}
}
