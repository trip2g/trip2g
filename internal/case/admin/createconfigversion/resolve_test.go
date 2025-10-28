package createconfigversion_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"trip2g/internal/case/admin/createconfigversion"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	"trip2g/internal/usertoken"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

type envMock = EnvMock

func assertConfigVersionPayload(t *testing.T, want, got model.CreateConfigVersionOrErrorPayload) {
	t.Helper()
	// Skip time comparison for CreatedAt field
	if payload, ok := got.(*model.CreateConfigVersionPayload); ok {
		if wantPayload, wantOk := want.(*model.CreateConfigVersionPayload); wantOk {
			require.Equal(t, wantPayload.ConfigVersion.ID, payload.ConfigVersion.ID)
			require.Equal(t, wantPayload.ConfigVersion.CreatedBy, payload.ConfigVersion.CreatedBy)
			require.Equal(t, wantPayload.ConfigVersion.ShowDraftVersions, payload.ConfigVersion.ShowDraftVersions)
			require.Equal(t, wantPayload.ConfigVersion.DefaultLayout, payload.ConfigVersion.DefaultLayout)
			require.Equal(t, wantPayload.ConfigVersion.Timezone, payload.ConfigVersion.Timezone)
			return
		}
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Resolve() = %v, want %v", got, want)
		for _, desc := range pretty.Diff(got, want) {
			t.Error(desc)
		}
	}
}

func TestResolve(t *testing.T) {
	type args struct {
		ctx   context.Context
		input model.CreateConfigVersionInput
	}

	tests := []struct {
		name          string
		env           createconfigversion.Env
		args          args
		want          model.CreateConfigVersionOrErrorPayload
		wantErr       bool
		afterCallback func(t *testing.T, mockEnv *envMock)
	}{
		{
			name: "successful config version creation",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 123}, nil
				},
				InsertConfigVersionFunc: func(ctx context.Context, params db.InsertConfigVersionParams) (db.ConfigVersion, error) {
					return db.ConfigVersion{
						ID:                1,
						CreatedBy:         params.CreatedBy,
						ShowDraftVersions: params.ShowDraftVersions,
						DefaultLayout:     params.DefaultLayout,
						Timezone:          params.Timezone,
						CreatedAt:         time.Now(),
					}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateConfigVersionInput{
					ShowDraftVersions: true,
					DefaultLayout:     "grid",
					Timezone:          "America/New_York",
				},
			},
			want: &model.CreateConfigVersionPayload{
				ConfigVersion: &db.ConfigVersion{
					ID:                1,
					CreatedBy:         123,
					ShowDraftVersions: true,
					DefaultLayout:     "grid",
					Timezone:          "America/New_York",
					CreatedAt:         time.Time{}, // will be set by test
				},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Len(t, mockEnv.InsertConfigVersionCalls(), 1)

				// Verify config version parameters
				params := mockEnv.InsertConfigVersionCalls()[0].Params
				require.Equal(t, int64(123), params.CreatedBy)
				require.Equal(t, true, params.ShowDraftVersions)
				require.Equal(t, "grid", params.DefaultLayout)
				require.Equal(t, "America/New_York", params.Timezone)
			},
		},
		{
			name: "successful config version creation with UTC timezone",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 456}, nil
				},
				InsertConfigVersionFunc: func(ctx context.Context, params db.InsertConfigVersionParams) (db.ConfigVersion, error) {
					return db.ConfigVersion{
						ID:                2,
						CreatedBy:         params.CreatedBy,
						ShowDraftVersions: params.ShowDraftVersions,
						DefaultLayout:     params.DefaultLayout,
						Timezone:          params.Timezone,
						CreatedAt:         time.Now(),
					}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateConfigVersionInput{
					ShowDraftVersions: false,
					DefaultLayout:     "list",
					Timezone:          "UTC",
				},
			},
			want: &model.CreateConfigVersionPayload{
				ConfigVersion: &db.ConfigVersion{
					ID:                2,
					CreatedBy:         456,
					ShowDraftVersions: false,
					DefaultLayout:     "list",
					Timezone:          "UTC",
					CreatedAt:         time.Time{},
				},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Len(t, mockEnv.InsertConfigVersionCalls(), 1)

				params := mockEnv.InsertConfigVersionCalls()[0].Params
				require.Equal(t, int64(456), params.CreatedBy)
				require.Equal(t, false, params.ShowDraftVersions)
				require.Equal(t, "list", params.DefaultLayout)
				require.Equal(t, "UTC", params.Timezone)
			},
		},
		{
			name: "validation error - missing timezone",
			env:  &envMock{},
			args: args{
				ctx: context.Background(),
				input: model.CreateConfigVersionInput{
					ShowDraftVersions: true,
					DefaultLayout:     "grid",
					Timezone:          "",
				},
			},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{
					{Name: "timezone", Value: "cannot be blank"},
				},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Empty(t, mockEnv.CurrentAdminUserTokenCalls())
				require.Empty(t, mockEnv.InsertConfigVersionCalls())
			},
		},
		{
			name: "successful config version creation with blank default layout",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 789}, nil
				},
				InsertConfigVersionFunc: func(ctx context.Context, params db.InsertConfigVersionParams) (db.ConfigVersion, error) {
					return db.ConfigVersion{
						ID:                3,
						CreatedBy:         params.CreatedBy,
						ShowDraftVersions: params.ShowDraftVersions,
						DefaultLayout:     params.DefaultLayout,
						Timezone:          params.Timezone,
						CreatedAt:         time.Now(),
					}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateConfigVersionInput{
					ShowDraftVersions: true,
					DefaultLayout:     "",
					Timezone:          "UTC",
				},
			},
			want: &model.CreateConfigVersionPayload{
				ConfigVersion: &db.ConfigVersion{
					ID:                3,
					CreatedBy:         789,
					ShowDraftVersions: true,
					DefaultLayout:     "",
					Timezone:          "UTC",
					CreatedAt:         time.Time{},
				},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Len(t, mockEnv.InsertConfigVersionCalls(), 1)

				params := mockEnv.InsertConfigVersionCalls()[0].Params
				require.Equal(t, int64(789), params.CreatedBy)
				require.Equal(t, true, params.ShowDraftVersions)
				require.Equal(t, "", params.DefaultLayout)
				require.Equal(t, "UTC", params.Timezone)
			},
		},
		{
			name: "validation error - invalid timezone",
			env:  &envMock{},
			args: args{
				ctx: context.Background(),
				input: model.CreateConfigVersionInput{
					ShowDraftVersions: true,
					DefaultLayout:     "grid",
					Timezone:          "Invalid/Timezone",
				},
			},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{
					{Name: "timezone", Value: "invalid timezone: unknown time zone Invalid/Timezone"},
				},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Empty(t, mockEnv.CurrentAdminUserTokenCalls())
				require.Empty(t, mockEnv.InsertConfigVersionCalls())
			},
		},
		{
			name: "validation error - missing timezone with blank default layout",
			env:  &envMock{},
			args: args{
				ctx: context.Background(),
				input: model.CreateConfigVersionInput{
					ShowDraftVersions: true,
					DefaultLayout:     "",
					Timezone:          "",
				},
			},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{
					{Name: "timezone", Value: "cannot be blank"},
				},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Empty(t, mockEnv.CurrentAdminUserTokenCalls())
				require.Empty(t, mockEnv.InsertConfigVersionCalls())
			},
		},
		{
			name: "error - admin token error",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return nil, errors.New("unauthorized")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateConfigVersionInput{
					ShowDraftVersions: true,
					DefaultLayout:     "grid",
					Timezone:          "UTC",
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Empty(t, mockEnv.InsertConfigVersionCalls())
			},
		},
		{
			name: "error - database insertion fails",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 789}, nil
				},
				InsertConfigVersionFunc: func(ctx context.Context, params db.InsertConfigVersionParams) (db.ConfigVersion, error) {
					return db.ConfigVersion{}, errors.New("database constraint violation")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateConfigVersionInput{
					ShowDraftVersions: true,
					DefaultLayout:     "grid",
					Timezone:          "UTC",
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Len(t, mockEnv.InsertConfigVersionCalls(), 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createconfigversion.Resolve(tt.args.ctx, tt.env, tt.args.input)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assertConfigVersionPayload(t, tt.want, got)

			if tt.afterCallback != nil {
				mockEnv := tt.env.(*envMock)
				tt.afterCallback(t, mockEnv)
			}
		})
	}
}

func TestValidateTimezone(t *testing.T) {
	tests := []struct {
		name     string
		timezone string
		wantErr  bool
	}{
		{
			name:     "valid UTC timezone",
			timezone: "UTC",
			wantErr:  false,
		},
		{
			name:     "valid America/New_York timezone",
			timezone: "America/New_York",
			wantErr:  false,
		},
		{
			name:     "valid Europe/London timezone",
			timezone: "Europe/London",
			wantErr:  false,
		},
		{
			name:     "valid Asia/Tokyo timezone",
			timezone: "Asia/Tokyo",
			wantErr:  false,
		},
		{
			name:     "invalid timezone",
			timezone: "Invalid/Timezone",
			wantErr:  true,
		},
		{
			name:     "random string timezone",
			timezone: "RandomString",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the timezone validation directly using time.LoadLocation
			_, err := time.LoadLocation(tt.timezone)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
