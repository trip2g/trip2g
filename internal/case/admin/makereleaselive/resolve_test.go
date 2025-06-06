package makereleaselive_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"trip2g/internal/case/admin/makereleaselive"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
	"trip2g/internal/usertoken"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg makereleaselive_test . Env

type Env interface {
	ReleaseByID(ctx context.Context, id int64) (db.Release, error)
	ChangeLiveRelease(ctx context.Context, id int64) error
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	PrepareLiveNotes(ctx context.Context) (*appmodel.NoteViews, error)
}

type envMock = EnvMock

func TestResolve(t *testing.T) {
	type args struct {
		ctx   context.Context
		input model.MakeReleaseLiveInput
	}
	tests := []struct {
		name          string
		env           makereleaselive.Env
		args          args
		want          model.MakeReleaseLiveOrErrorPayload
		wantErr       bool
		afterCallback func(t *testing.T, mockEnv *envMock)
	}{
		{
			name: "successful make release live",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				ReleaseByIDFunc: func(ctx context.Context, id int64) (db.Release, error) {
					return db.Release{
						ID:        123,
						Title:     "Test Release",
						CreatedBy: 1,
						IsLive:    false,
					}, nil
				},
				ChangeLiveReleaseFunc: func(ctx context.Context, id int64) error {
					return nil
				},
				PrepareLiveNotesFunc: func(ctx context.Context) (*appmodel.NoteViews, error) {
					return &appmodel.NoteViews{}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.MakeReleaseLiveInput{
					ID: 123,
				},
			},
			want: &model.MakeReleaseLivePayload{
				Release: &db.Release{
					ID:        123,
					Title:     "Test Release",
					CreatedBy: 1,
					IsLive:    false,
				},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Len(t, mockEnv.ReleaseByIDCalls(), 1)
				require.Len(t, mockEnv.ChangeLiveReleaseCalls(), 1)
				require.Len(t, mockEnv.PrepareLiveNotesCalls(), 1)

				// Verify correct ID was passed
				require.Equal(t, int64(123), mockEnv.ReleaseByIDCalls()[0].ID)
				require.Equal(t, int64(123), mockEnv.ChangeLiveReleaseCalls()[0].ID)
			},
		},
		{
			name: "admin token error",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return nil, errors.New("unauthorized")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.MakeReleaseLiveInput{
					ID: 123,
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				// Other methods should not be called due to early error
				require.Empty(t, mockEnv.ReleaseByIDCalls())
				require.Empty(t, mockEnv.ChangeLiveReleaseCalls())
				require.Empty(t, mockEnv.PrepareLiveNotesCalls())
			},
		},
		{
			name: "release not found",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				ReleaseByIDFunc: func(ctx context.Context, id int64) (db.Release, error) {
					return db.Release{}, errors.New("release not found")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.MakeReleaseLiveInput{
					ID: 999,
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Len(t, mockEnv.ReleaseByIDCalls(), 1)
				// Later methods should not be called due to error
				require.Empty(t, mockEnv.ChangeLiveReleaseCalls())
				require.Empty(t, mockEnv.PrepareLiveNotesCalls())

				// Verify correct ID was passed
				require.Equal(t, int64(999), mockEnv.ReleaseByIDCalls()[0].ID)
			},
		},
		{
			name: "change live release error",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				ReleaseByIDFunc: func(ctx context.Context, id int64) (db.Release, error) {
					return db.Release{
						ID:        123,
						Title:     "Test Release",
						CreatedBy: 1,
						IsLive:    false,
					}, nil
				},
				ChangeLiveReleaseFunc: func(ctx context.Context, id int64) error {
					return errors.New("database error")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.MakeReleaseLiveInput{
					ID: 123,
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Len(t, mockEnv.ReleaseByIDCalls(), 1)
				require.Len(t, mockEnv.ChangeLiveReleaseCalls(), 1)
				// PrepareNotes should not be called due to error
				require.Empty(t, mockEnv.PrepareLiveNotesCalls())

				// Verify correct IDs were passed
				require.Equal(t, int64(123), mockEnv.ReleaseByIDCalls()[0].ID)
				require.Equal(t, int64(123), mockEnv.ChangeLiveReleaseCalls()[0].ID)
			},
		},
		{
			name: "prepare live notes error",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 1}, nil
				},
				ReleaseByIDFunc: func(ctx context.Context, id int64) (db.Release, error) {
					return db.Release{
						ID:        123,
						Title:     "Test Release",
						CreatedBy: 1,
						IsLive:    false,
					}, nil
				},
				ChangeLiveReleaseFunc: func(ctx context.Context, id int64) error {
					return nil
				},
				PrepareLiveNotesFunc: func(ctx context.Context) (*appmodel.NoteViews, error) {
					return nil, errors.New("failed to prepare notes")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.MakeReleaseLiveInput{
					ID: 123,
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Len(t, mockEnv.ReleaseByIDCalls(), 1)
				require.Len(t, mockEnv.ChangeLiveReleaseCalls(), 1)
				require.Len(t, mockEnv.PrepareLiveNotesCalls(), 1)

				// Verify correct IDs were passed
				require.Equal(t, int64(123), mockEnv.ReleaseByIDCalls()[0].ID)
				require.Equal(t, int64(123), mockEnv.ChangeLiveReleaseCalls()[0].ID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := makereleaselive.Resolve(tt.args.ctx, tt.env, tt.args.input)
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

			// Verify env method calls using afterCallback
			if tt.afterCallback != nil {
				mockEnv := tt.env.(*envMock)
				tt.afterCallback(t, mockEnv)
			}
		})
	}
}
