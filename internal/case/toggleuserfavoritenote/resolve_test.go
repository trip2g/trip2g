package toggleuserfavoritenote_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"

	"trip2g/internal/case/toggleuserfavoritenote"
	"trip2g/internal/db"
	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
	"trip2g/internal/usertoken"
)

type envMock = EnvMock

func TestResolve(t *testing.T) {
	ctx := context.Background()

	// Mock data
	validUserToken := &usertoken.Data{ID: 123}
	mockNote := &appmodel.NoteView{
		PathID:    456,
		VersionID: 789,
	}
	mockNoteViews := &appmodel.NoteViews{
		Map: map[string]*appmodel.NoteView{
			"note456": mockNote,
		},
	}

	tests := []struct {
		name string
		env  *envMock
		args struct {
			ctx   context.Context
			input model.ToggleFavoriteNoteInput
		}
		want          model.ToggleFavoriteNoteOrErrorPayload
		wantErr       bool
		afterCallback func(t *testing.T, env *envMock)
	}{
		{
			name: "success - add to favorites",
			env: &envMock{
				CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return validUserToken, nil
				},
				LiveNoteViewsFunc: func() *appmodel.NoteViews {
					return mockNoteViews
				},
				InsertUserFavoriteNoteFunc: func(ctx context.Context, arg db.InsertUserFavoriteNoteParams) error {
					require.Equal(t, int64(123), arg.UserID)
					require.Equal(t, int64(789), arg.NoteVersionID)
					return nil
				},
			},
			args: struct {
				ctx   context.Context
				input model.ToggleFavoriteNoteInput
			}{
				ctx: ctx,
				input: model.ToggleFavoriteNoteInput{
					PathID: 456,
					Value:  true,
				},
			},
			want: &model.ToggleFavoriteNotePayload{
				Success: true,
				UserID:  123,
			},
			wantErr: false,
			afterCallback: func(t *testing.T, env *envMock) {
				require.Len(t, env.CurrentUserTokenCalls(), 1)
				require.Len(t, env.LiveNoteViewsCalls(), 1)
				require.Len(t, env.InsertUserFavoriteNoteCalls(), 1)
				require.Empty(t, env.DeleteUserFavoriteNoteCalls())
			},
		},
		{
			name: "success - remove from favorites",
			env: &envMock{
				CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return validUserToken, nil
				},
				LiveNoteViewsFunc: func() *appmodel.NoteViews {
					return mockNoteViews
				},
				DeleteUserFavoriteNoteFunc: func(ctx context.Context, arg db.DeleteUserFavoriteNoteParams) error {
					require.Equal(t, int64(123), arg.UserID)
					require.Equal(t, int64(789), arg.NoteVersionID)
					return nil
				},
			},
			args: struct {
				ctx   context.Context
				input model.ToggleFavoriteNoteInput
			}{
				ctx: ctx,
				input: model.ToggleFavoriteNoteInput{
					PathID: 456,
					Value:  false,
				},
			},
			want: &model.ToggleFavoriteNotePayload{
				Success: true,
				UserID:  123,
			},
			wantErr: false,
			afterCallback: func(t *testing.T, env *envMock) {
				require.Len(t, env.CurrentUserTokenCalls(), 1)
				require.Len(t, env.LiveNoteViewsCalls(), 1)
				require.Empty(t, env.InsertUserFavoriteNoteCalls())
				require.Len(t, env.DeleteUserFavoriteNoteCalls(), 1)
			},
		},
		{
			name: "validation error - missing path ID",
			env:  &envMock{},
			args: struct {
				ctx   context.Context
				input model.ToggleFavoriteNoteInput
			}{
				ctx: ctx,
				input: model.ToggleFavoriteNoteInput{
					PathID: 0,
					Value:  true,
				},
			},
			want: &model.ErrorPayload{
				ByFields: []model.FieldMessage{
					{
						Name:  "pathId",
						Value: "cannot be blank",
					},
				},
			},
			wantErr: false,
			afterCallback: func(t *testing.T, env *envMock) {
				require.Empty(t, env.CurrentUserTokenCalls())
			},
		},
		{
			name: "error - current user token error",
			env: &envMock{
				CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return nil, errors.New("token error")
				},
			},
			args: struct {
				ctx   context.Context
				input model.ToggleFavoriteNoteInput
			}{
				ctx: ctx,
				input: model.ToggleFavoriteNoteInput{
					PathID: 456,
					Value:  true,
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "error - no auth",
			env: &envMock{
				CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return nil, nil
				},
			},
			args: struct {
				ctx   context.Context
				input model.ToggleFavoriteNoteInput
			}{
				ctx: ctx,
				input: model.ToggleFavoriteNoteInput{
					PathID: 456,
					Value:  true,
				},
			},
			want: &model.ErrorPayload{
				Message: "no auth",
			},
			wantErr: false,
			afterCallback: func(t *testing.T, env *envMock) {
				require.Len(t, env.CurrentUserTokenCalls(), 1)
			},
		},
		{
			name: "error - note not found",
			env: &envMock{
				CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return validUserToken, nil
				},
				LiveNoteViewsFunc: func() *appmodel.NoteViews {
					return &appmodel.NoteViews{} // Empty, no notes
				},
			},
			args: struct {
				ctx   context.Context
				input model.ToggleFavoriteNoteInput
			}{
				ctx: ctx,
				input: model.ToggleFavoriteNoteInput{
					PathID: 999, // Non-existent
					Value:  true,
				},
			},
			want: &model.ErrorPayload{
				Message: "note not found",
			},
			wantErr: false,
			afterCallback: func(t *testing.T, env *envMock) {
				require.Len(t, env.CurrentUserTokenCalls(), 1)
				require.Len(t, env.LiveNoteViewsCalls(), 1)
			},
		},
		{
			name: "error - insert favorite fails",
			env: &envMock{
				CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return validUserToken, nil
				},
				LiveNoteViewsFunc: func() *appmodel.NoteViews {
					return mockNoteViews
				},
				InsertUserFavoriteNoteFunc: func(ctx context.Context, arg db.InsertUserFavoriteNoteParams) error {
					return errors.New("database error")
				},
			},
			args: struct {
				ctx   context.Context
				input model.ToggleFavoriteNoteInput
			}{
				ctx: ctx,
				input: model.ToggleFavoriteNoteInput{
					PathID: 456,
					Value:  true,
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "error - delete favorite fails",
			env: &envMock{
				CurrentUserTokenFunc: func(ctx context.Context) (*usertoken.Data, error) {
					return validUserToken, nil
				},
				LiveNoteViewsFunc: func() *appmodel.NoteViews {
					return mockNoteViews
				},
				DeleteUserFavoriteNoteFunc: func(ctx context.Context, arg db.DeleteUserFavoriteNoteParams) error {
					return errors.New("database error")
				},
			},
			args: struct {
				ctx   context.Context
				input model.ToggleFavoriteNoteInput
			}{
				ctx: ctx,
				input: model.ToggleFavoriteNoteInput{
					PathID: 456,
					Value:  false,
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toggleuserfavoritenote.Resolve(tt.args.ctx, tt.env, tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				diff := pretty.Diff(got, tt.want)
				t.Errorf("Resolve() diff: %v", diff)
			}
			if tt.afterCallback != nil {
				tt.afterCallback(t, tt.env)
			}
		})
	}
}
