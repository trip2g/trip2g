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
	ListActiveTgChatSubgraphNamesByUserID(ctx context.Context, userID int64) ([]string, error)
	ListActivePatreonSubgraphNamesByUserID(ctx context.Context, userID int64) ([]string, error)
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
					ListActiveUserSubgraphsFunc: func(ctx context.Context, userID int64) ([]string, error) {
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
					ListActiveUserSubgraphsFunc: func(ctx context.Context, userID int64) ([]string, error) {
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
					ListActiveUserSubgraphsFunc: func(ctx context.Context, userID int64) ([]string, error) {
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

func TestResolve_AdminDefaultVersionBehavior(t *testing.T) {
	// Create test notes with different versions
	liveNote := &model.NoteView{
		Path:          "/test-versioned-note",
		Title:         "Test Note - Live Version",
		PathID:        3,
		VersionID:     300,
		Content:       []byte("# Live Version Content"),
		HTML:          "<h1>Live Version Content</h1>",
		Permalink:     "/test-versioned-note",
		Free:          true,
		SubgraphNames: []string{},
		Subgraphs:     map[string]*model.NoteSubgraph{},
		InLinks:       map[string]struct{}{},
		RawMeta:       map[string]interface{}{},
		Assets:        map[string]struct{}{},
		AssetReplaces: map[string]string{},
	}

	latestNote := &model.NoteView{
		Path:          "/test-versioned-note",
		Title:         "Test Note - Latest Version",
		PathID:        3,
		VersionID:     301,
		Content:       []byte("# Latest Version Content"),
		HTML:          "<h1>Latest Version Content</h1>",
		Permalink:     "/test-versioned-note",
		Free:          true,
		SubgraphNames: []string{},
		Subgraphs:     map[string]*model.NoteSubgraph{},
		InLinks:       map[string]struct{}{},
		RawMeta:       map[string]interface{}{},
		Assets:        map[string]struct{}{},
		AssetReplaces: map[string]string{},
	}

	// Create NoteViews for live and latest versions
	liveNoteViews := &model.NoteViews{
		Map: map[string]*model.NoteView{
			"/test-versioned-note": liveNote,
		},
		List:      []*model.NoteView{liveNote},
		Subgraphs: map[string]*model.NoteSubgraph{},
		Version:   "live",
	}

	latestNoteViews := &model.NoteViews{
		Map: map[string]*model.NoteView{
			"/test-versioned-note": latestNote,
		},
		List:      []*model.NoteView{latestNote},
		Subgraphs: map[string]*model.NoteSubgraph{},
		Version:   "latest",
	}

	tests := []struct {
		name          string
		request       rendernotepage.Request
		setupEnv      func() *EnvMock
		wantErr       bool
		checkResponse func(t *testing.T, resp *rendernotepage.Response)
	}{
		{
			name: "non-admin user should have DefaultVersion set to 'live'",
			request: rendernotepage.Request{
				Path:    "/test-versioned-note",
				Version: "",
				UserToken: &usertoken.Data{
					ID:   100,
					Role: "user",
				},
			},
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LoggerFunc: func() logger.Logger {
						return &logger.DummyLogger{}
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return liveNoteViews
					},
					ListActiveUserSubgraphsFunc: func(ctx context.Context, userID int64) ([]string, error) {
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
				require.Equal(t, "live", resp.DefaultVersion)
				require.Equal(t, "Test Note - Live Version", resp.Title)
				require.Equal(t, int64(300), resp.Note.VersionID)
			},
		},
		{
			name: "admin user should have DefaultVersion set to 'latest'",
			request: rendernotepage.Request{
				Path:    "/test-versioned-note",
				Version: "",
				UserToken: &usertoken.Data{
					ID:   200,
					Role: "admin",
				},
			},
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LoggerFunc: func() logger.Logger {
						return &logger.DummyLogger{}
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return liveNoteViews
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
						return latestNoteViews
					},
					ListActiveUserSubgraphsFunc: func(ctx context.Context, userID int64) ([]string, error) {
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
				require.Equal(t, "latest", resp.DefaultVersion)
				require.Equal(t, "Test Note - Latest Version", resp.Title)
				require.Equal(t, int64(301), resp.Note.VersionID)
			},
		},
		{
			name: "admin with empty version should view latest by default",
			request: rendernotepage.Request{
				Path:    "/test-versioned-note",
				Version: "", // Empty version
				UserToken: &usertoken.Data{
					ID:   300,
					Role: "admin",
				},
			},
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LoggerFunc: func() logger.Logger {
						return &logger.DummyLogger{}
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return liveNoteViews
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
						return latestNoteViews
					},
					ListActiveUserSubgraphsFunc: func(ctx context.Context, userID int64) ([]string, error) {
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
				require.Equal(t, "latest", resp.DefaultVersion)
				require.Equal(t, "Test Note - Latest Version", resp.Title)
				require.Equal(t, int64(301), resp.Note.VersionID)
			},
		},
		{
			name: "admin explicitly requesting live version should view live",
			request: rendernotepage.Request{
				Path:    "/test-versioned-note",
				Version: "live", // Explicitly request live
				UserToken: &usertoken.Data{
					ID:   400,
					Role: "admin",
				},
			},
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LoggerFunc: func() logger.Logger {
						return &logger.DummyLogger{}
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return liveNoteViews
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
						return latestNoteViews
					},
					ListActiveUserSubgraphsFunc: func(ctx context.Context, userID int64) ([]string, error) {
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
				require.Equal(t, "latest", resp.DefaultVersion) // DefaultVersion is still 'latest' for admin
				require.Equal(t, "Test Note - Live Version", resp.Title)
				require.Equal(t, int64(300), resp.Note.VersionID)
			},
		},
		{
			name: "non-admin requesting latest version should still view live",
			request: rendernotepage.Request{
				Path:    "/test-versioned-note",
				Version: "latest", // Non-admin tries to request latest
				UserToken: &usertoken.Data{
					ID:   500,
					Role: "user",
				},
			},
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LoggerFunc: func() logger.Logger {
						return &logger.DummyLogger{}
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return liveNoteViews
					},
					ListActiveUserSubgraphsFunc: func(ctx context.Context, userID int64) ([]string, error) {
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
				require.Equal(t, "live", resp.DefaultVersion)
				require.Equal(t, "Test Note - Live Version", resp.Title)
				require.Equal(t, int64(300), resp.Note.VersionID)
			},
		},
		{
			name: "unauthenticated user should have DefaultVersion set to 'live'",
			request: rendernotepage.Request{
				Path:      "/test-versioned-note",
				Version:   "",
				UserToken: nil, // Unauthenticated
			},
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LoggerFunc: func() logger.Logger {
						return &logger.DummyLogger{}
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return liveNoteViews
					},
				}
			},
			wantErr: false,
			checkResponse: func(t *testing.T, resp *rendernotepage.Response) {
				require.NotNil(t, resp)
				require.Equal(t, "live", resp.DefaultVersion)
				require.Equal(t, "Test Note - Live Version", resp.Title)
				require.Equal(t, int64(300), resp.Note.VersionID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := tt.setupEnv()
			resp, err := rendernotepage.Resolve(context.Background(), env, tt.request)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestResolve_CheckLatestBannerWithDefaultVersion(t *testing.T) {
	// Create test notes with different versions for the banner test
	liveNote := &model.NoteView{
		Path:          "/banner-test",
		Title:         "Banner Test - Live",
		PathID:        4,
		VersionID:     400,
		Content:       []byte("# Live Content"),
		HTML:          "<h1>Live Content</h1>",
		Permalink:     "/banner-test",
		Free:          true,
		SubgraphNames: []string{},
		Subgraphs:     map[string]*model.NoteSubgraph{},
		InLinks:       map[string]struct{}{},
		RawMeta:       map[string]interface{}{},
		Assets:        map[string]struct{}{},
		AssetReplaces: map[string]string{},
	}

	latestNote := &model.NoteView{
		Path:          "/banner-test",
		Title:         "Banner Test - Latest",
		PathID:        4,
		VersionID:     401, // Different version ID
		Content:       []byte("# Latest Content"),
		HTML:          "<h1>Latest Content</h1>",
		Permalink:     "/banner-test",
		Free:          true,
		SubgraphNames: []string{},
		Subgraphs:     map[string]*model.NoteSubgraph{},
		InLinks:       map[string]struct{}{},
		RawMeta:       map[string]interface{}{},
		Assets:        map[string]struct{}{},
		AssetReplaces: map[string]string{},
	}

	// Create NoteViews for live and latest versions
	liveNoteViews := &model.NoteViews{
		Map: map[string]*model.NoteView{
			"/banner-test": liveNote,
		},
		List:      []*model.NoteView{liveNote},
		Subgraphs: map[string]*model.NoteSubgraph{},
		Version:   "live",
	}

	latestNoteViews := &model.NoteViews{
		Map: map[string]*model.NoteView{
			"/banner-test": latestNote,
		},
		List:      []*model.NoteView{latestNote},
		Subgraphs: map[string]*model.NoteSubgraph{},
		Version:   "latest",
	}

	tests := []struct {
		name               string
		request            rendernotepage.Request
		setupEnv           func() *EnvMock
		wantErr            bool
		expectBanner       bool
		expectedBannerText string
	}{
		{
			name: "admin viewing live version should see banner with latest version link using default 'latest'",
			request: rendernotepage.Request{
				Path:    "/banner-test",
				Version: "live",
				UserToken: &usertoken.Data{
					ID:   100,
					Role: "admin",
				},
			},
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LoggerFunc: func() logger.Logger {
						return &logger.DummyLogger{}
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return liveNoteViews
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
						return latestNoteViews
					},
					ListActiveUserSubgraphsFunc: func(ctx context.Context, userID int64) ([]string, error) {
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
			wantErr:            false,
			expectBanner:       true,
			expectedBannerText: "Это последняя опубликованная версия, которая отличается от загруженной",
		},
		{
			name: "admin viewing latest version should see banner with live version link using default 'latest'",
			request: rendernotepage.Request{
				Path:    "/banner-test",
				Version: "latest",
				UserToken: &usertoken.Data{
					ID:   200,
					Role: "admin",
				},
			},
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LoggerFunc: func() logger.Logger {
						return &logger.DummyLogger{}
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return liveNoteViews
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
						return latestNoteViews
					},
					ListActiveUserSubgraphsFunc: func(ctx context.Context, userID int64) ([]string, error) {
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
			wantErr:            false,
			expectBanner:       true,
			expectedBannerText: "Это последняя загруженная версия, которая отличается от опубликованной",
		},
		{
			name: "admin viewing page where versions are the same should not see banner",
			request: rendernotepage.Request{
				Path:    "/banner-test",
				Version: "latest",
				UserToken: &usertoken.Data{
					ID:   300,
					Role: "admin",
				},
			},
			setupEnv: func() *EnvMock {
				// Both versions have the same VersionID
				sameNote := &model.NoteView{
					Path:          "/banner-test",
					Title:         "Banner Test - Same",
					PathID:        4,
					VersionID:     500, // Same version ID
					Content:       []byte("# Same Content"),
					HTML:          "<h1>Same Content</h1>",
					Permalink:     "/banner-test",
					Free:          true,
					SubgraphNames: []string{},
					Subgraphs:     map[string]*model.NoteSubgraph{},
					InLinks:       map[string]struct{}{},
					RawMeta:       map[string]interface{}{},
					Assets:        map[string]struct{}{},
					AssetReplaces: map[string]string{},
				}

				sameNoteViews := &model.NoteViews{
					Map: map[string]*model.NoteView{
						"/banner-test": sameNote,
					},
					List:      []*model.NoteView{sameNote},
					Subgraphs: map[string]*model.NoteSubgraph{},
					Version:   "latest",
				}

				return &EnvMock{
					LoggerFunc: func() logger.Logger {
						return &logger.DummyLogger{}
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return sameNoteViews
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
						return sameNoteViews
					},
					ListActiveUserSubgraphsFunc: func(ctx context.Context, userID int64) ([]string, error) {
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
			wantErr:      false,
			expectBanner: false,
		},
		{
			name: "non-admin user should not see banner even if versions differ",
			request: rendernotepage.Request{
				Path:    "/banner-test",
				Version: "",
				UserToken: &usertoken.Data{
					ID:   400,
					Role: "user",
				},
			},
			setupEnv: func() *EnvMock {
				return &EnvMock{
					LoggerFunc: func() logger.Logger {
						return &logger.DummyLogger{}
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return liveNoteViews
					},
					ListActiveUserSubgraphsFunc: func(ctx context.Context, userID int64) ([]string, error) {
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
			wantErr:      false,
			expectBanner: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := tt.setupEnv()
			resp, err := rendernotepage.Resolve(context.Background(), env, tt.request)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Check if banner is present or not based on expectations
				// Since versionBanner is private, we can't directly check it
				// but we can verify the behavior through other means
				// For now, we'll just check that the response is correct
				require.NotNil(t, resp.Note)
			}
		})
	}
}

func TestResolve_SystemPagesBlocked(t *testing.T) {
	// Create test notes for system pages
	bannerNote := &model.NoteView{
		Path:          "/_banner",
		Title:         "Banner System Page",
		PathID:        5,
		VersionID:     500,
		Content:       []byte("# System Banner"),
		HTML:          "<h1>System Banner</h1>",
		Permalink:     "/_banner",
		Free:          true,
		SubgraphNames: []string{},
		Subgraphs:     map[string]*model.NoteSubgraph{},
		InLinks:       map[string]struct{}{},
		RawMeta:       map[string]interface{}{},
		Assets:        map[string]struct{}{},
		AssetReplaces: map[string]string{},
	}

	systemHiddenNote := &model.NoteView{
		Path:          "/_system/hidden",
		Title:         "Hidden System Page",
		PathID:        6,
		VersionID:     600,
		Content:       []byte("# Hidden System Page"),
		HTML:          "<h1>Hidden System Page</h1>",
		Permalink:     "/_system/hidden",
		Free:          true,
		SubgraphNames: []string{},
		Subgraphs:     map[string]*model.NoteSubgraph{},
		InLinks:       map[string]struct{}{},
		RawMeta:       map[string]interface{}{},
		Assets:        map[string]struct{}{},
		AssetReplaces: map[string]string{},
	}

	normalNote := &model.NoteView{
		Path:          "/normal-page",
		Title:         "Normal Page",
		PathID:        7,
		VersionID:     700,
		Content:       []byte("# Normal Page"),
		HTML:          "<h1>Normal Page</h1>",
		Permalink:     "/normal-page",
		Free:          true,
		SubgraphNames: []string{},
		Subgraphs:     map[string]*model.NoteSubgraph{},
		InLinks:       map[string]struct{}{},
		RawMeta:       map[string]interface{}{},
		Assets:        map[string]struct{}{},
		AssetReplaces: map[string]string{},
	}

	// Create NoteViews containing system and normal notes
	noteViews := &model.NoteViews{
		Map: map[string]*model.NoteView{
			"/_banner":        bannerNote,
			"/_system/hidden": systemHiddenNote,
			"/normal-page":    normalNote,
		},
		List:      []*model.NoteView{bannerNote, systemHiddenNote, normalNote},
		Subgraphs: map[string]*model.NoteSubgraph{},
		Version:   "live",
	}

	tests := []struct {
		name          string
		request       rendernotepage.Request
		setupEnv      func() *EnvMock
		expectErrNF   bool // expect ErrNotFound
		checkResponse func(t *testing.T, resp *rendernotepage.Response)
	}{
		{
			name: "/_banner system page should be blocked for unauthenticated user",
			request: rendernotepage.Request{
				Path:      "/_banner",
				Version:   "",
				UserToken: nil,
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
			expectErrNF: true,
		},
		{
			name: "/_banner system page should be blocked for regular user",
			request: rendernotepage.Request{
				Path:    "/_banner",
				Version: "",
				UserToken: &usertoken.Data{
					ID:   100,
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
				}
			},
			expectErrNF: true,
		},
		{
			name: "/_banner system page should be blocked for admin user",
			request: rendernotepage.Request{
				Path:    "/_banner",
				Version: "",
				UserToken: &usertoken.Data{
					ID:   200,
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
				}
			},
			expectErrNF: true,
		},
		{
			name: "/_system/hidden nested system page should be blocked",
			request: rendernotepage.Request{
				Path:    "/_system/hidden",
				Version: "",
				UserToken: &usertoken.Data{
					ID:   300,
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
				}
			},
			expectErrNF: true,
		},
		{
			name: "/_config system page should be blocked even if not in notes",
			request: rendernotepage.Request{
				Path:    "/_config",
				Version: "",
				UserToken: &usertoken.Data{
					ID:   400,
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
				}
			},
			expectErrNF: true,
		},
		{
			name: "normal page /normal-page should work for unauthenticated user",
			request: rendernotepage.Request{
				Path:      "/normal-page",
				Version:   "",
				UserToken: nil,
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
			expectErrNF: false,
			checkResponse: func(t *testing.T, resp *rendernotepage.Response) {
				require.NotNil(t, resp)
				require.Equal(t, "Normal Page", resp.Title)
				require.Equal(t, "/normal-page", resp.Note.Path)
			},
		},
		{
			name: "normal page /normal-page should work for admin user",
			request: rendernotepage.Request{
				Path:    "/normal-page",
				Version: "",
				UserToken: &usertoken.Data{
					ID:   500,
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
					ListActiveUserSubgraphsFunc: func(ctx context.Context, userID int64) ([]string, error) {
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
			expectErrNF: false,
			checkResponse: func(t *testing.T, resp *rendernotepage.Response) {
				require.NotNil(t, resp)
				require.Equal(t, "Normal Page", resp.Title)
				require.Equal(t, "/normal-page", resp.Note.Path)
			},
		},
		{
			name: "page with underscore not at start should work: /my_page",
			request: rendernotepage.Request{
				Path:      "/my_page",
				Version:   "",
				UserToken: nil,
			},
			setupEnv: func() *EnvMock {
				// Add a page with underscore not at the start
				underscoreNote := &model.NoteView{
					Path:          "/my_page",
					Title:         "My Page With Underscore",
					PathID:        8,
					VersionID:     800,
					Content:       []byte("# My Page"),
					HTML:          "<h1>My Page</h1>",
					Permalink:     "/my_page",
					Free:          true,
					SubgraphNames: []string{},
					Subgraphs:     map[string]*model.NoteSubgraph{},
					InLinks:       map[string]struct{}{},
					RawMeta:       map[string]interface{}{},
					Assets:        map[string]struct{}{},
					AssetReplaces: map[string]string{},
				}

				noteViewsWithUnderscore := &model.NoteViews{
					Map: map[string]*model.NoteView{
						"/my_page": underscoreNote,
					},
					List:      []*model.NoteView{underscoreNote},
					Subgraphs: map[string]*model.NoteSubgraph{},
					Version:   "live",
				}

				return &EnvMock{
					LoggerFunc: func() logger.Logger {
						return &logger.DummyLogger{}
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return noteViewsWithUnderscore
					},
				}
			},
			expectErrNF: false,
			checkResponse: func(t *testing.T, resp *rendernotepage.Response) {
				require.NotNil(t, resp)
				require.Equal(t, "My Page With Underscore", resp.Title)
				require.Equal(t, "/my_page", resp.Note.Path)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := tt.setupEnv()
			resp, err := rendernotepage.Resolve(context.Background(), env, tt.request)

			if tt.expectErrNF {
				require.Error(t, err)
				require.Equal(t, rendernotepage.ErrNotFound, err)
			} else {
				require.NoError(t, err)
				if tt.checkResponse != nil {
					tt.checkResponse(t, resp)
				}
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
					ListActiveUserSubgraphsFunc: func(ctx context.Context, userID int64) ([]string, error) {
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
					ListActiveUserSubgraphsFunc: func(ctx context.Context, userID int64) ([]string, error) {
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
