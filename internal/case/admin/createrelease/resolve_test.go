package createrelease_test

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	"trip2g/internal/case/admin/createrelease"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
	"trip2g/internal/usertoken"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

//go:generate go tool github.com/matryer/moq -out mocks_test.go -pkg createrelease_test . Env

func assertReleasePayload(t *testing.T, want, got model.CreateReleaseOrErrorPayload) {
	t.Helper()
	// Skip time comparison for CreatedAt field
	if payload, ok := got.(*model.CreateReleasePayload); ok {
		if wantPayload, wantOk := want.(*model.CreateReleasePayload); wantOk {
			require.Equal(t, wantPayload.Release.ID, payload.Release.ID)
			require.Equal(t, wantPayload.Release.Title, payload.Release.Title)
			require.Equal(t, wantPayload.Release.CreatedBy, payload.Release.CreatedBy)
			require.Equal(t, wantPayload.Release.HomeNoteVersionID, payload.Release.HomeNoteVersionID)
			require.Equal(t, wantPayload.Release.IsLive, payload.Release.IsLive)
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

type Env interface {
	InsertRelease(ctx context.Context, arg db.InsertReleaseParams) (db.Release, error)
	InsertReleaseNoteVersion(ctx context.Context, arg db.InsertReleaseNoteVersionParams) error
	ChangeLiveRelease(ctx context.Context, id int64) error
	LatestNoteViews() *appmodel.NoteViews
	CurrentAdminUserToken(ctx context.Context) (*usertoken.Data, error)
	PrepareLiveNotes(ctx context.Context) (*appmodel.NoteViews, error)
}

type envMock = EnvMock

func TestResolve(t *testing.T) {
	type args struct {
		ctx   context.Context
		input model.CreateReleaseInput
	}

	tests := []struct {
		name          string
		env           createrelease.Env
		args          args
		want          model.CreateReleaseOrErrorPayload
		wantErr       bool
		afterCallback func(t *testing.T, mockEnv *envMock)
	}{
		{
			name: "successful release creation without home note",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 123}, nil
				},
				LatestNoteViewsFunc: func() *appmodel.NoteViews {
					return &appmodel.NoteViews{
						List: []*appmodel.NoteView{
							{VersionID: 1, Path: "note1.md"},
							{VersionID: 2, Path: "note2.md"},
						},
					}
				},
				InsertReleaseFunc: func(ctx context.Context, arg db.InsertReleaseParams) (db.Release, error) {
					return db.Release{
						ID:                1,
						Title:             arg.Title,
						CreatedBy:         arg.CreatedBy,
						HomeNoteVersionID: arg.HomeNoteVersionID,
						CreatedAt:         time.Now(),
						IsLive:            true,
					}, nil
				},
				ChangeLiveReleaseFunc: func(ctx context.Context, id int64) error {
					return nil
				},
				InsertReleaseNoteVersionFunc: func(ctx context.Context, arg db.InsertReleaseNoteVersionParams) error {
					return nil
				},
				PrepareLiveNotesFunc: func(ctx context.Context) (*appmodel.NoteViews, error) {
					return &appmodel.NoteViews{}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateReleaseInput{
					Title:             "  Test Release  ", // will be normalized
					HomeNoteVersionID: nil,
				},
			},
			want: &model.CreateReleasePayload{
				Release: &db.Release{
					ID:                1,
					Title:             "test release", // normalized
					CreatedBy:         123,
					HomeNoteVersionID: sql.NullInt64{Valid: false},
					CreatedAt:         time.Time{}, // will be set by test
					IsLive:            true,
				},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Len(t, mockEnv.LatestNoteViewsCalls(), 1)
				require.Len(t, mockEnv.InsertReleaseCalls(), 1)
				require.Len(t, mockEnv.ChangeLiveReleaseCalls(), 1)
				require.Len(t, mockEnv.InsertReleaseNoteVersionCalls(), 2) // 2 note views
				require.Len(t, mockEnv.PrepareLiveNotesCalls(), 1)

				// Verify release parameters
				releaseParams := mockEnv.InsertReleaseCalls()[0].Arg
				require.Equal(t, "test release", releaseParams.Title)
				require.Equal(t, int64(123), releaseParams.CreatedBy)
				require.False(t, releaseParams.HomeNoteVersionID.Valid)

				// Verify live release change
				require.Equal(t, int64(1), mockEnv.ChangeLiveReleaseCalls()[0].ID)

				// Verify note version insertions
				require.Equal(t, int64(1), mockEnv.InsertReleaseNoteVersionCalls()[0].Arg.NoteVersionID)
				require.Equal(t, int64(1), mockEnv.InsertReleaseNoteVersionCalls()[0].Arg.ReleaseID)
				require.Equal(t, int64(2), mockEnv.InsertReleaseNoteVersionCalls()[1].Arg.NoteVersionID)
				require.Equal(t, int64(1), mockEnv.InsertReleaseNoteVersionCalls()[1].Arg.ReleaseID)
			},
		},
		{
			name: "successful release creation with home note",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 456}, nil
				},
				LatestNoteViewsFunc: func() *appmodel.NoteViews {
					return &appmodel.NoteViews{
						List: []*appmodel.NoteView{
							{VersionID: 10, Path: "index.md"},
							{VersionID: 20, Path: "about.md"},
						},
					}
				},
				InsertReleaseFunc: func(ctx context.Context, arg db.InsertReleaseParams) (db.Release, error) {
					return db.Release{
						ID:                2,
						Title:             arg.Title,
						CreatedBy:         arg.CreatedBy,
						HomeNoteVersionID: arg.HomeNoteVersionID,
						CreatedAt:         time.Now(),
						IsLive:            true,
					}, nil
				},
				ChangeLiveReleaseFunc: func(ctx context.Context, id int64) error {
					return nil
				},
				InsertReleaseNoteVersionFunc: func(ctx context.Context, arg db.InsertReleaseNoteVersionParams) error {
					return nil
				},
				PrepareLiveNotesFunc: func(ctx context.Context) (*appmodel.NoteViews, error) {
					return &appmodel.NoteViews{}, nil
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateReleaseInput{
					Title:             "Release with Home",
					HomeNoteVersionID: int64Ptr(10),
				},
			},
			want: &model.CreateReleasePayload{
				Release: &db.Release{
					ID:                2,
					Title:             "release with home",
					CreatedBy:         456,
					HomeNoteVersionID: sql.NullInt64{Int64: 10, Valid: true},
					CreatedAt:         time.Time{},
					IsLive:            true,
				},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				releaseParams := mockEnv.InsertReleaseCalls()[0].Arg
				require.True(t, releaseParams.HomeNoteVersionID.Valid)
				require.Equal(t, int64(10), releaseParams.HomeNoteVersionID.Int64)
			},
		},
		{
			name: "error - invalid home note version ID",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 123}, nil
				},
				LatestNoteViewsFunc: func() *appmodel.NoteViews {
					return &appmodel.NoteViews{
						List: []*appmodel.NoteView{
							{VersionID: 1, Path: "note1.md"},
							{VersionID: 2, Path: "note2.md"},
						},
					}
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateReleaseInput{
					Title:             "Test Release",
					HomeNoteVersionID: int64Ptr(999), // not in note views
				},
			},
			want:    &model.ErrorPayload{Message: "home note version ID does not exist in latest note views"},
			wantErr: false,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Len(t, mockEnv.LatestNoteViewsCalls(), 1)
				require.Empty(t, mockEnv.InsertReleaseCalls()) // should not insert
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
				input: model.CreateReleaseInput{
					Title: "Test Release",
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.CurrentAdminUserTokenCalls(), 1)
				require.Empty(t, mockEnv.LatestNoteViewsCalls())
			},
		},
		{
			name: "error - insert release fails",
			env: &envMock{
				CurrentAdminUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return &usertoken.Data{ID: 123}, nil
				},
				LatestNoteViewsFunc: func() *appmodel.NoteViews {
					return &appmodel.NoteViews{
						List: []*appmodel.NoteView{
							{VersionID: 1, Path: "note1.md"},
						},
					}
				},
				InsertReleaseFunc: func(ctx context.Context, arg db.InsertReleaseParams) (db.Release, error) {
					return db.Release{}, errors.New("database error")
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.CreateReleaseInput{
					Title: "Test Release",
				},
			},
			want:    nil,
			wantErr: true,
			afterCallback: func(t *testing.T, mockEnv *envMock) {
				require.Len(t, mockEnv.InsertReleaseCalls(), 1)
				require.Empty(t, mockEnv.ChangeLiveReleaseCalls())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createrelease.Resolve(tt.args.ctx, tt.env, tt.args.input)
			
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			
			require.NoError(t, err)
			assertReleasePayload(t, tt.want, got)

			if tt.afterCallback != nil {
				mockEnv := tt.env.(*envMock)
				tt.afterCallback(t, mockEnv)
			}
		})
	}
}

func int64Ptr(i int64) *int64 {
	return &i
}
