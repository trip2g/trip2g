package rendernotepage_test

import (
	"context"
	"testing"

	"trip2g/internal/case/rendernotepage"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
)

//go:generate go run github.com/matryer/moq -out mocks_test.go -pkg rendernotepage_test . Env

type Env interface {
	Logger() logger.Logger
	LatestNoteViews() *model.NoteViews
	LiveNoteViews() *model.NoteViews
	ListActiveSubgraphNamesByUserID(ctx context.Context, userID int64) ([]string, error)
	InsertUserNoteView(ctx context.Context, params db.InsertUserNoteViewParams) error
	UpsertUserNoteDailyView(ctx context.Context, params db.UpsertUserNoteDailyViewParams) (int64, error)
	IncreaseUserNoteViewCount(ctx context.Context, userID int64) error
}

func TestResolve_FreeNoteWithSubgraph(t *testing.T) {
	// Create a test note that is free but has subgraphs
	testNote := &model.NoteView{
		Path:          "/test-free-note",
		Title:         "Test Free Note",
		PathID:        1,
		VersionID:     100,
		Content:       []byte("# Test Free Note Content"),
		HTML:          "<h1>Test Free Note Content</h1>",
		Permalink:     "/test-free-note",
		Free:          true,                // Note is free
		SubgraphNames: []string{"premium"}, // But has subgraph
		Subgraphs: map[string]*model.NoteSubgraph{
			"premium": {
				Name: "premium",
			},
		},
		InLinks: map[string]struct{}{},
		RawMeta: map[string]interface{}{
			"free":     true,
			"subgraph": "premium",
		},
		Assets:        map[string]struct{}{},
		AssetReplaces: map[string]string{},
	}

	// Create NoteViews containing the test note
	noteViews := &model.NoteViews{
		Map: map[string]*model.NoteView{
			"/test-free-note": testNote,
		},
		List: []*model.NoteView{testNote},
		Subgraphs: map[string]*model.NoteSubgraph{
			"premium": {
				Name: "premium",
			},
		},
		Version: "live",
	}

	tests := []struct {
		name          string
		request       rendernotepage.Request
		setupEnv      func() *EnvMock
		wantErr       bool
		expectedError error
		checkResponse func(t *testing.T, resp *rendernotepage.Response)
	}{
		{
			name: "free note with subgraph should render for unauthenticated user",
			request: rendernotepage.Request{
				Path:      "/test-free-note",
				Version:   "",
				Referrer:  "",
				UserToken: nil, // Unauthenticated user
			},
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LoggerFunc: func() logger.Logger {
						return &logger.DummyLogger{}
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
				}
			},
			wantErr: false,
			checkResponse: func(t *testing.T, resp *rendernotepage.Response) {
				require.NotNil(t, resp)
				require.Equal(t, "Test Free Note", resp.Title)
				require.NotNil(t, resp.Note)
				require.Equal(t, "/test-free-note", resp.Note.Path)
				require.True(t, resp.Note.Free)
				require.Equal(t, []string{"premium"}, resp.Note.SubgraphNames)
				require.Nil(t, resp.UserToken)
			},
		},
		{
			name: "free note with subgraph should render for authenticated user without subgraph access",
			request: rendernotepage.Request{
				Path:     "/test-free-note",
				Version:  "",
				Referrer: "",
				UserToken: &usertoken.Data{
					ID:   123,
					Role: "user",
				},
			},
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LoggerFunc: func() logger.Logger {
						return &logger.DummyLogger{}
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					ListActiveSubgraphNamesByUserIDFunc: func(ctx context.Context, userID int64) ([]string, error) {
						// User has no subgraph access
						return []string{}, nil
					},
					InsertUserNoteViewFunc: func(ctx context.Context, params db.InsertUserNoteViewParams) error {
						return nil
					},
					UpsertUserNoteDailyViewFunc: func(ctx context.Context, params db.UpsertUserNoteDailyViewParams) (int64, error) {
						return 1, nil
					},
					IncreaseUserNoteViewCountFunc: func(ctx context.Context, userID int64) error {
						return nil
					},
				}
			},
			wantErr: false,
			checkResponse: func(t *testing.T, resp *rendernotepage.Response) {
				require.NotNil(t, resp)
				require.Equal(t, "Test Free Note", resp.Title)
				require.NotNil(t, resp.Note)
				require.Equal(t, "/test-free-note", resp.Note.Path)
				require.True(t, resp.Note.Free)
				require.Equal(t, []string{}, resp.UserSubgraphs)
			},
		},
		{
			name: "free note with subgraph should render for authenticated user with subgraph access",
			request: rendernotepage.Request{
				Path:     "/test-free-note",
				Version:  "",
				Referrer: "",
				UserToken: &usertoken.Data{
					ID:   456,
					Role: "user",
				},
			},
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LoggerFunc: func() logger.Logger {
						return &logger.DummyLogger{}
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					ListActiveSubgraphNamesByUserIDFunc: func(ctx context.Context, userID int64) ([]string, error) {
						// User has premium subgraph access
						return []string{"premium"}, nil
					},
					InsertUserNoteViewFunc: func(ctx context.Context, params db.InsertUserNoteViewParams) error {
						return nil
					},
					UpsertUserNoteDailyViewFunc: func(ctx context.Context, params db.UpsertUserNoteDailyViewParams) (int64, error) {
						return 1, nil
					},
					IncreaseUserNoteViewCountFunc: func(ctx context.Context, userID int64) error {
						return nil
					},
				}
			},
			wantErr: false,
			checkResponse: func(t *testing.T, resp *rendernotepage.Response) {
				require.NotNil(t, resp)
				require.Equal(t, "Test Free Note", resp.Title)
				require.NotNil(t, resp.Note)
				require.Equal(t, "/test-free-note", resp.Note.Path)
				require.True(t, resp.Note.Free)
				require.Equal(t, []string{"premium"}, resp.UserSubgraphs)
			},
		},
		{
			name: "free note with subgraph should render for admin user",
			request: rendernotepage.Request{
				Path:     "/test-free-note",
				Version:  "",
				Referrer: "",
				UserToken: &usertoken.Data{
					ID:   789,
					Role: "admin",
				},
			},
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LoggerFunc: func() logger.Logger {
						return &logger.DummyLogger{}
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					ListActiveSubgraphNamesByUserIDFunc: func(ctx context.Context, userID int64) ([]string, error) {
						// Admin might not have explicit subgraph access
						return []string{}, nil
					},
					InsertUserNoteViewFunc: func(ctx context.Context, params db.InsertUserNoteViewParams) error {
						return nil
					},
					UpsertUserNoteDailyViewFunc: func(ctx context.Context, params db.UpsertUserNoteDailyViewParams) (int64, error) {
						return 1, nil
					},
					IncreaseUserNoteViewCountFunc: func(ctx context.Context, userID int64) error {
						return nil
					},
				}
			},
			wantErr: false,
			checkResponse: func(t *testing.T, resp *rendernotepage.Response) {
				require.NotNil(t, resp)
				require.Equal(t, "Test Free Note", resp.Title)
				require.NotNil(t, resp.Note)
				require.Equal(t, "/test-free-note", resp.Note.Path)
				require.True(t, resp.Note.Free)
				// Admin should have access regardless of subgraph
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := tt.setupEnv()
			resp, err := rendernotepage.Resolve(context.Background(), env, tt.request)

			if tt.wantErr {
				require.Error(t, err)
				if tt.expectedError != nil {
					require.Equal(t, tt.expectedError, err)
				}
			} else {
				require.NoError(t, err)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestResolve_NonFreeNoteWithSubgraph(t *testing.T) {
	// Create a test note that is NOT free and has subgraphs
	testNote := &model.NoteView{
		Path:          "/test-paid-note",
		Title:         "Test Paid Note",
		PathID:        2,
		VersionID:     200,
		Content:       []byte("# Test Paid Note Content"),
		HTML:          "<h1>Test Paid Note Content</h1>",
		Permalink:     "/test-paid-note",
		Free:          false,               // Note is NOT free
		SubgraphNames: []string{"premium"}, // And has subgraph
		Subgraphs: map[string]*model.NoteSubgraph{
			"premium": {
				Name: "premium",
			},
		},
		InLinks: map[string]struct{}{},
		RawMeta: map[string]interface{}{
			"subgraph": "premium",
		},
		Assets:        map[string]struct{}{},
		AssetReplaces: map[string]string{},
	}

	// Create NoteViews containing the test note
	noteViews := &model.NoteViews{
		Map: map[string]*model.NoteView{
			"/test-paid-note": testNote,
		},
		List: []*model.NoteView{testNote},
		Subgraphs: map[string]*model.NoteSubgraph{
			"premium": {
				Name: "premium",
			},
		},
		Version: "live",
	}

	tests := []struct {
		name          string
		request       rendernotepage.Request
		setupEnv      func() *EnvMock
		wantErr       bool
		expectedError error
	}{
		{
			name: "non-free note with subgraph should show paywall for unauthenticated user",
			request: rendernotepage.Request{
				Path:      "/test-paid-note",
				Version:   "",
				Referrer:  "",
				UserToken: nil, // Unauthenticated user
			},
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LoggerFunc: func() logger.Logger {
						return &logger.DummyLogger{}
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
				}
			},
			wantErr:       true,
			expectedError: &rendernotepage.PaywallError{Message: "Need auth"},
		},
		{
			name: "non-free note with subgraph should show paywall for authenticated user without subgraph access",
			request: rendernotepage.Request{
				Path:     "/test-paid-note",
				Version:  "",
				Referrer: "",
				UserToken: &usertoken.Data{
					ID:   123,
					Role: "user",
				},
			},
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LoggerFunc: func() logger.Logger {
						return &logger.DummyLogger{}
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					ListActiveSubgraphNamesByUserIDFunc: func(ctx context.Context, userID int64) ([]string, error) {
						// User has no subgraph access
						return []string{}, nil
					},
					InsertUserNoteViewFunc: func(ctx context.Context, params db.InsertUserNoteViewParams) error {
						return nil
					},
					UpsertUserNoteDailyViewFunc: func(ctx context.Context, params db.UpsertUserNoteDailyViewParams) (int64, error) {
						return 1, nil
					},
					IncreaseUserNoteViewCountFunc: func(ctx context.Context, userID int64) error {
						return nil
					},
				}
			},
			wantErr:       true,
			expectedError: &rendernotepage.PaywallError{Message: "Need subscription"},
		},
		{
			name: "non-free note with subgraph should render for authenticated user with subgraph access",
			request: rendernotepage.Request{
				Path:     "/test-paid-note",
				Version:  "",
				Referrer: "",
				UserToken: &usertoken.Data{
					ID:   456,
					Role: "user",
				},
			},
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LoggerFunc: func() logger.Logger {
						return &logger.DummyLogger{}
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					ListActiveSubgraphNamesByUserIDFunc: func(ctx context.Context, userID int64) ([]string, error) {
						// User has premium subgraph access
						return []string{"premium"}, nil
					},
					InsertUserNoteViewFunc: func(ctx context.Context, params db.InsertUserNoteViewParams) error {
						return nil
					},
					UpsertUserNoteDailyViewFunc: func(ctx context.Context, params db.UpsertUserNoteDailyViewParams) (int64, error) {
						return 1, nil
					},
					IncreaseUserNoteViewCountFunc: func(ctx context.Context, userID int64) error {
						return nil
					},
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := tt.setupEnv()
			_, err := rendernotepage.Resolve(context.Background(), env, tt.request)

			if tt.wantErr {
				require.Error(t, err)
				if tt.expectedError != nil {
					require.EqualError(t, err, tt.expectedError.Error())
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
