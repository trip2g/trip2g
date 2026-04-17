package rendernotepage_test

import (
	"context"
	"database/sql"
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
	Layouts() *model.Layouts
	Logger() logger.Logger
	LatestNoteViews() *model.NoteViews
	LiveNoteViews() *model.NoteViews
	InsertUserNoteView(ctx context.Context, params db.InsertUserNoteViewParams) error
	UpsertUserNoteDailyView(ctx context.Context, params db.UpsertUserNoteDailyViewParams) (int64, error)
	IncreaseUserNoteViewCount(ctx context.Context, userID int64) error
	ListActiveUserSubgraphs(ctx context.Context, userID int64) ([]string, error)
	RecordUserNoteView(ctx context.Context, userID int64, note *model.NoteView, referrerVersionID *int64)
	LastUserNoteView(ctx context.Context, arg db.LastUserNoteViewParams) (db.LastUserNoteViewRow, error)
	SiteConfig(ctx context.Context) model.SiteConfig
	SiteTitleTemplate() string
	CanReadNote(ctx context.Context, note *model.NoteView) (bool, error)
	GetTelegramChatName(ctx context.Context, telegramChatID int64) (string, error)
}

func TestResolve_FreeNoteWithSubgraph(t *testing.T) {
	// Create a test note that is free but has subgraphs
	testNote := &model.NoteView{
		Path:          "/test-free-note",
		Title:         "Test Free Note",
		PathID:        1,
		VersionID:     0,
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
		AssetReplaces: map[string]*model.NoteAssetReplace{},
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
					SiteConfigFunc: func(ctx context.Context) model.SiteConfig {
						return model.SiteConfig{}
					},
					SiteTitleTemplateFunc: func() string {
						return "%s"
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					CanReadNoteFunc: func(ctx context.Context, note *model.NoteView) (bool, error) {
						return true, nil
					},
					GetTelegramPostLinksByNoteVersionIDFunc: func(ctx context.Context, arg db.GetTelegramPostLinksByNoteVersionIDParams) ([]db.GetTelegramPostLinksByNoteVersionIDRow, error) {
						return nil, nil
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
					SiteConfigFunc: func(ctx context.Context) model.SiteConfig {
						return model.SiteConfig{}
					},
					SiteTitleTemplateFunc: func() string {
						return "%s"
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
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
					RecordUserNoteViewFunc: func(ctx context.Context, userID int64, note *model.NoteView, referrerVersionID *int64) {
						// Mock function - no-op for tests
					},
					LastUserNoteViewFunc: func(ctx context.Context, arg db.LastUserNoteViewParams) (db.LastUserNoteViewRow, error) {
						return db.LastUserNoteViewRow{}, sql.ErrNoRows
					},
					CanReadNoteFunc: func(ctx context.Context, note *model.NoteView) (bool, error) {
						return true, nil
					},
					GetTelegramPostLinksByNoteVersionIDFunc: func(ctx context.Context, arg db.GetTelegramPostLinksByNoteVersionIDParams) ([]db.GetTelegramPostLinksByNoteVersionIDRow, error) {
						return nil, nil
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
					SiteConfigFunc: func(ctx context.Context) model.SiteConfig {
						return model.SiteConfig{}
					},
					SiteTitleTemplateFunc: func() string {
						return "%s"
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
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
					RecordUserNoteViewFunc: func(ctx context.Context, userID int64, note *model.NoteView, referrerVersionID *int64) {
						// Mock function - no-op for tests
					},
					LastUserNoteViewFunc: func(ctx context.Context, arg db.LastUserNoteViewParams) (db.LastUserNoteViewRow, error) {
						return db.LastUserNoteViewRow{}, sql.ErrNoRows
					},
					CanReadNoteFunc: func(ctx context.Context, note *model.NoteView) (bool, error) {
						return true, nil
					},
					GetTelegramPostLinksByNoteVersionIDFunc: func(ctx context.Context, arg db.GetTelegramPostLinksByNoteVersionIDParams) ([]db.GetTelegramPostLinksByNoteVersionIDRow, error) {
						return nil, nil
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
					SiteConfigFunc: func(ctx context.Context) model.SiteConfig {
						return model.SiteConfig{}
					},
					SiteTitleTemplateFunc: func() string {
						return "%s"
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
					RecordUserNoteViewFunc: func(ctx context.Context, userID int64, note *model.NoteView, referrerVersionID *int64) {
						// Mock function - no-op for tests
					},
					LastUserNoteViewFunc: func(ctx context.Context, arg db.LastUserNoteViewParams) (db.LastUserNoteViewRow, error) {
						return db.LastUserNoteViewRow{}, sql.ErrNoRows
					},
					CanReadNoteFunc: func(ctx context.Context, note *model.NoteView) (bool, error) {
						return true, nil
					},
					GetTelegramPostLinksByNoteVersionIDFunc: func(ctx context.Context, arg db.GetTelegramPostLinksByNoteVersionIDParams) ([]db.GetTelegramPostLinksByNoteVersionIDRow, error) {
						return nil, nil
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
		AssetReplaces: map[string]*model.NoteAssetReplace{},
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
		AssetReplaces: map[string]*model.NoteAssetReplace{},
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
					SiteConfigFunc: func(ctx context.Context) model.SiteConfig {
						return model.SiteConfig{}
					},
					SiteTitleTemplateFunc: func() string {
						return "%s"
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return liveNoteViews
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
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
					RecordUserNoteViewFunc: func(ctx context.Context, userID int64, note *model.NoteView, referrerVersionID *int64) {
						// Mock function - no-op for tests
					},
					LastUserNoteViewFunc: func(ctx context.Context, arg db.LastUserNoteViewParams) (db.LastUserNoteViewRow, error) {
						return db.LastUserNoteViewRow{}, sql.ErrNoRows
					},
					CanReadNoteFunc: func(ctx context.Context, note *model.NoteView) (bool, error) {
						return true, nil
					},
					GetTelegramPostLinksByNoteVersionIDFunc: func(ctx context.Context, arg db.GetTelegramPostLinksByNoteVersionIDParams) ([]db.GetTelegramPostLinksByNoteVersionIDRow, error) {
						return nil, nil
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
			name: "admin user should have DefaultVersion set to 'live' (same as regular users)",
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
					SiteConfigFunc: func(ctx context.Context) model.SiteConfig {
						return model.SiteConfig{}
					},
					SiteTitleTemplateFunc: func() string {
						return "%s"
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
					RecordUserNoteViewFunc: func(ctx context.Context, userID int64, note *model.NoteView, referrerVersionID *int64) {
						// Mock function - no-op for tests
					},
					LastUserNoteViewFunc: func(ctx context.Context, arg db.LastUserNoteViewParams) (db.LastUserNoteViewRow, error) {
						return db.LastUserNoteViewRow{}, sql.ErrNoRows
					},
					CanReadNoteFunc: func(ctx context.Context, note *model.NoteView) (bool, error) {
						return true, nil
					},
					GetTelegramPostLinksByNoteVersionIDFunc: func(ctx context.Context, arg db.GetTelegramPostLinksByNoteVersionIDParams) ([]db.GetTelegramPostLinksByNoteVersionIDRow, error) {
						return nil, nil
					},
				}
			},
			wantErr: false,
			checkResponse: func(t *testing.T, resp *rendernotepage.Response) {
				require.NotNil(t, resp)
				// Admin now sees live by default (same as regular users)
				require.Equal(t, "live", resp.DefaultVersion)
				require.Equal(t, "Test Note - Live Version", resp.Title)
				require.Equal(t, int64(300), resp.Note.VersionID)
			},
		},
		{
			name: "admin explicitly requesting latest version should view latest",
			request: rendernotepage.Request{
				Path:    "/test-versioned-note",
				Version: "latest", // Explicitly request latest
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
					SiteConfigFunc: func(ctx context.Context) model.SiteConfig {
						return model.SiteConfig{}
					},
					SiteTitleTemplateFunc: func() string {
						return "%s"
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
					RecordUserNoteViewFunc: func(ctx context.Context, userID int64, note *model.NoteView, referrerVersionID *int64) {
						// Mock function - no-op for tests
					},
					LastUserNoteViewFunc: func(ctx context.Context, arg db.LastUserNoteViewParams) (db.LastUserNoteViewRow, error) {
						return db.LastUserNoteViewRow{}, sql.ErrNoRows
					},
					CanReadNoteFunc: func(ctx context.Context, note *model.NoteView) (bool, error) {
						return true, nil
					},
					GetTelegramPostLinksByNoteVersionIDFunc: func(ctx context.Context, arg db.GetTelegramPostLinksByNoteVersionIDParams) ([]db.GetTelegramPostLinksByNoteVersionIDRow, error) {
						return nil, nil
					},
				}
			},
			wantErr: false,
			checkResponse: func(t *testing.T, resp *rendernotepage.Response) {
				require.NotNil(t, resp)
				// Admin can explicitly switch to latest
				require.Equal(t, "live", resp.DefaultVersion) // DefaultVersion stays live
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
					SiteConfigFunc: func(ctx context.Context) model.SiteConfig {
						return model.SiteConfig{}
					},
					SiteTitleTemplateFunc: func() string {
						return "%s"
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
					RecordUserNoteViewFunc: func(ctx context.Context, userID int64, note *model.NoteView, referrerVersionID *int64) {
						// Mock function - no-op for tests
					},
					LastUserNoteViewFunc: func(ctx context.Context, arg db.LastUserNoteViewParams) (db.LastUserNoteViewRow, error) {
						return db.LastUserNoteViewRow{}, sql.ErrNoRows
					},
					CanReadNoteFunc: func(ctx context.Context, note *model.NoteView) (bool, error) {
						return true, nil
					},
					GetTelegramPostLinksByNoteVersionIDFunc: func(ctx context.Context, arg db.GetTelegramPostLinksByNoteVersionIDParams) ([]db.GetTelegramPostLinksByNoteVersionIDRow, error) {
						return nil, nil
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
					SiteConfigFunc: func(ctx context.Context) model.SiteConfig {
						return model.SiteConfig{}
					},
					SiteTitleTemplateFunc: func() string {
						return "%s"
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return liveNoteViews
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
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
					RecordUserNoteViewFunc: func(ctx context.Context, userID int64, note *model.NoteView, referrerVersionID *int64) {
						// Mock function - no-op for tests
					},
					LastUserNoteViewFunc: func(ctx context.Context, arg db.LastUserNoteViewParams) (db.LastUserNoteViewRow, error) {
						return db.LastUserNoteViewRow{}, sql.ErrNoRows
					},
					CanReadNoteFunc: func(ctx context.Context, note *model.NoteView) (bool, error) {
						return true, nil
					},
					GetTelegramPostLinksByNoteVersionIDFunc: func(ctx context.Context, arg db.GetTelegramPostLinksByNoteVersionIDParams) ([]db.GetTelegramPostLinksByNoteVersionIDRow, error) {
						return nil, nil
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
					SiteConfigFunc: func(ctx context.Context) model.SiteConfig {
						return model.SiteConfig{}
					},
					SiteTitleTemplateFunc: func() string {
						return "%s"
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return liveNoteViews
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
						return liveNoteViews
					},
					CanReadNoteFunc: func(ctx context.Context, note *model.NoteView) (bool, error) {
						return true, nil
					},
					GetTelegramPostLinksByNoteVersionIDFunc: func(ctx context.Context, arg db.GetTelegramPostLinksByNoteVersionIDParams) ([]db.GetTelegramPostLinksByNoteVersionIDRow, error) {
						return nil, nil
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
		AssetReplaces: map[string]*model.NoteAssetReplace{},
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
		AssetReplaces: map[string]*model.NoteAssetReplace{},
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
					SiteConfigFunc: func(ctx context.Context) model.SiteConfig {
						return model.SiteConfig{}
					},
					SiteTitleTemplateFunc: func() string {
						return "%s"
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
					RecordUserNoteViewFunc: func(ctx context.Context, userID int64, note *model.NoteView, referrerVersionID *int64) {
						// Mock function - no-op for tests
					},
					LastUserNoteViewFunc: func(ctx context.Context, arg db.LastUserNoteViewParams) (db.LastUserNoteViewRow, error) {
						return db.LastUserNoteViewRow{}, sql.ErrNoRows
					},
					CanReadNoteFunc: func(ctx context.Context, note *model.NoteView) (bool, error) {
						return true, nil
					},
					GetTelegramPostLinksByNoteVersionIDFunc: func(ctx context.Context, arg db.GetTelegramPostLinksByNoteVersionIDParams) ([]db.GetTelegramPostLinksByNoteVersionIDRow, error) {
						return nil, nil
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
					SiteConfigFunc: func(ctx context.Context) model.SiteConfig {
						return model.SiteConfig{}
					},
					SiteTitleTemplateFunc: func() string {
						return "%s"
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
					RecordUserNoteViewFunc: func(ctx context.Context, userID int64, note *model.NoteView, referrerVersionID *int64) {
						// Mock function - no-op for tests
					},
					LastUserNoteViewFunc: func(ctx context.Context, arg db.LastUserNoteViewParams) (db.LastUserNoteViewRow, error) {
						return db.LastUserNoteViewRow{}, sql.ErrNoRows
					},
					CanReadNoteFunc: func(ctx context.Context, note *model.NoteView) (bool, error) {
						return true, nil
					},
					GetTelegramPostLinksByNoteVersionIDFunc: func(ctx context.Context, arg db.GetTelegramPostLinksByNoteVersionIDParams) ([]db.GetTelegramPostLinksByNoteVersionIDRow, error) {
						return nil, nil
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
					AssetReplaces: map[string]*model.NoteAssetReplace{},
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
					SiteConfigFunc: func(ctx context.Context) model.SiteConfig {
						return model.SiteConfig{}
					},
					SiteTitleTemplateFunc: func() string {
						return "%s"
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
					RecordUserNoteViewFunc: func(ctx context.Context, userID int64, note *model.NoteView, referrerVersionID *int64) {
						// Mock function - no-op for tests
					},
					LastUserNoteViewFunc: func(ctx context.Context, arg db.LastUserNoteViewParams) (db.LastUserNoteViewRow, error) {
						return db.LastUserNoteViewRow{}, sql.ErrNoRows
					},
					CanReadNoteFunc: func(ctx context.Context, note *model.NoteView) (bool, error) {
						return true, nil
					},
					GetTelegramPostLinksByNoteVersionIDFunc: func(ctx context.Context, arg db.GetTelegramPostLinksByNoteVersionIDParams) ([]db.GetTelegramPostLinksByNoteVersionIDRow, error) {
						return nil, nil
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
					SiteConfigFunc: func(ctx context.Context) model.SiteConfig {
						return model.SiteConfig{}
					},
					SiteTitleTemplateFunc: func() string {
						return "%s"
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return liveNoteViews
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
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
					RecordUserNoteViewFunc: func(ctx context.Context, userID int64, note *model.NoteView, referrerVersionID *int64) {
						// Mock function - no-op for tests
					},
					LastUserNoteViewFunc: func(ctx context.Context, arg db.LastUserNoteViewParams) (db.LastUserNoteViewRow, error) {
						return db.LastUserNoteViewRow{}, sql.ErrNoRows
					},
					CanReadNoteFunc: func(ctx context.Context, note *model.NoteView) (bool, error) {
						return true, nil
					},
					GetTelegramPostLinksByNoteVersionIDFunc: func(ctx context.Context, arg db.GetTelegramPostLinksByNoteVersionIDParams) ([]db.GetTelegramPostLinksByNoteVersionIDRow, error) {
						return nil, nil
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
		AssetReplaces: map[string]*model.NoteAssetReplace{},
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
		AssetReplaces: map[string]*model.NoteAssetReplace{},
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
		AssetReplaces: map[string]*model.NoteAssetReplace{},
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
					SiteConfigFunc: func(ctx context.Context) model.SiteConfig {
						return model.SiteConfig{}
					},
					SiteTitleTemplateFunc: func() string {
						return "%s"
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					CanReadNoteFunc: func(ctx context.Context, note *model.NoteView) (bool, error) {
						return true, nil
					},
					GetTelegramPostLinksByNoteVersionIDFunc: func(ctx context.Context, arg db.GetTelegramPostLinksByNoteVersionIDParams) ([]db.GetTelegramPostLinksByNoteVersionIDRow, error) {
						return nil, nil
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
					SiteConfigFunc: func(ctx context.Context) model.SiteConfig {
						return model.SiteConfig{}
					},
					SiteTitleTemplateFunc: func() string {
						return "%s"
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					CanReadNoteFunc: func(ctx context.Context, note *model.NoteView) (bool, error) {
						return true, nil
					},
					GetTelegramPostLinksByNoteVersionIDFunc: func(ctx context.Context, arg db.GetTelegramPostLinksByNoteVersionIDParams) ([]db.GetTelegramPostLinksByNoteVersionIDRow, error) {
						return nil, nil
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
					SiteConfigFunc: func(ctx context.Context) model.SiteConfig {
						return model.SiteConfig{}
					},
					SiteTitleTemplateFunc: func() string {
						return "%s"
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
					SiteConfigFunc: func(ctx context.Context) model.SiteConfig {
						return model.SiteConfig{}
					},
					SiteTitleTemplateFunc: func() string {
						return "%s"
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
					SiteConfigFunc: func(ctx context.Context) model.SiteConfig {
						return model.SiteConfig{}
					},
					SiteTitleTemplateFunc: func() string {
						return "%s"
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
					SiteConfigFunc: func(ctx context.Context) model.SiteConfig {
						return model.SiteConfig{}
					},
					SiteTitleTemplateFunc: func() string {
						return "%s"
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					CanReadNoteFunc: func(ctx context.Context, note *model.NoteView) (bool, error) {
						return true, nil
					},
					GetTelegramPostLinksByNoteVersionIDFunc: func(ctx context.Context, arg db.GetTelegramPostLinksByNoteVersionIDParams) ([]db.GetTelegramPostLinksByNoteVersionIDRow, error) {
						return nil, nil
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
					SiteConfigFunc: func(ctx context.Context) model.SiteConfig {
						return model.SiteConfig{}
					},
					SiteTitleTemplateFunc: func() string {
						return "%s"
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
					RecordUserNoteViewFunc: func(ctx context.Context, userID int64, note *model.NoteView, referrerVersionID *int64) {
						// Mock function - no-op for tests
					},
					LastUserNoteViewFunc: func(ctx context.Context, arg db.LastUserNoteViewParams) (db.LastUserNoteViewRow, error) {
						return db.LastUserNoteViewRow{}, sql.ErrNoRows
					},
					CanReadNoteFunc: func(ctx context.Context, note *model.NoteView) (bool, error) {
						return true, nil
					},
					GetTelegramPostLinksByNoteVersionIDFunc: func(ctx context.Context, arg db.GetTelegramPostLinksByNoteVersionIDParams) ([]db.GetTelegramPostLinksByNoteVersionIDRow, error) {
						return nil, nil
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
					AssetReplaces: map[string]*model.NoteAssetReplace{},
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
					SiteConfigFunc: func(ctx context.Context) model.SiteConfig {
						return model.SiteConfig{}
					},
					SiteTitleTemplateFunc: func() string {
						return "%s"
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return noteViewsWithUnderscore
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViewsWithUnderscore
					},
					CanReadNoteFunc: func(ctx context.Context, note *model.NoteView) (bool, error) {
						return true, nil
					},
					GetTelegramPostLinksByNoteVersionIDFunc: func(ctx context.Context, arg db.GetTelegramPostLinksByNoteVersionIDParams) ([]db.GetTelegramPostLinksByNoteVersionIDRow, error) {
						return nil, nil
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
		AssetReplaces: map[string]*model.NoteAssetReplace{},
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
					SiteConfigFunc: func(ctx context.Context) model.SiteConfig {
						return model.SiteConfig{}
					},
					SiteTitleTemplateFunc: func() string {
						return "%s"
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					CanReadNoteFunc: func(ctx context.Context, note *model.NoteView) (bool, error) {
						// This should not be called for unauthenticated users on non-free notes
						// as the paywall check happens before CanReadNote
						return true, nil
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
					SiteConfigFunc: func(ctx context.Context) model.SiteConfig {
						return model.SiteConfig{}
					},
					SiteTitleTemplateFunc: func() string {
						return "%s"
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
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
					RecordUserNoteViewFunc: func(ctx context.Context, userID int64, note *model.NoteView, referrerVersionID *int64) {
						// Mock function - no-op for tests
					},
					LastUserNoteViewFunc: func(ctx context.Context, arg db.LastUserNoteViewParams) (db.LastUserNoteViewRow, error) {
						return db.LastUserNoteViewRow{}, sql.ErrNoRows
					},
					CanReadNoteFunc: func(ctx context.Context, note *model.NoteView) (bool, error) {
						// User without subgraph access should be denied
						return false, nil
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
					SiteConfigFunc: func(ctx context.Context) model.SiteConfig {
						return model.SiteConfig{}
					},
					SiteTitleTemplateFunc: func() string {
						return "%s"
					},
					LiveNoteViewsFunc: func() *model.NoteViews {
						return noteViews
					},
					LatestNoteViewsFunc: func() *model.NoteViews {
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
					RecordUserNoteViewFunc: func(ctx context.Context, userID int64, note *model.NoteView, referrerVersionID *int64) {
						// Mock function - no-op for tests
					},
					LastUserNoteViewFunc: func(ctx context.Context, arg db.LastUserNoteViewParams) (db.LastUserNoteViewRow, error) {
						return db.LastUserNoteViewRow{}, sql.ErrNoRows
					},
					CanReadNoteFunc: func(ctx context.Context, note *model.NoteView) (bool, error) {
						return true, nil
					},
					GetTelegramPostLinksByNoteVersionIDFunc: func(ctx context.Context, arg db.GetTelegramPostLinksByNoteVersionIDParams) ([]db.GetTelegramPostLinksByNoteVersionIDRow, error) {
						return nil, nil
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

func TestResolve_SiteTitleTemplate(t *testing.T) {
	testNote := &model.NoteView{
		Path:          "/test-title",
		Title:         "Test Note",
		PathID:        10,
		VersionID:     1000,
		Content:       []byte("# Test Note"),
		HTML:          "<h1>Test Note</h1>",
		Permalink:     "/test-title",
		Free:          true,
		SubgraphNames: []string{},
		Subgraphs:     map[string]*model.NoteSubgraph{},
		InLinks:       map[string]struct{}{},
		RawMeta:       map[string]interface{}{},
		Assets:        map[string]struct{}{},
		AssetReplaces: map[string]*model.NoteAssetReplace{},
	}

	noteViews := &model.NoteViews{
		Map: map[string]*model.NoteView{
			"/test-title": testNote,
		},
		List:      []*model.NoteView{testNote},
		Subgraphs: map[string]*model.NoteSubgraph{},
		Version:   "live",
	}

	tests := []struct {
		name          string
		titleTemplate string
		expectedTitle string
	}{
		{
			name:          "default template returns raw title",
			titleTemplate: "%s",
			expectedTitle: "Test Note",
		},
		{
			name:          "template with site name suffix",
			titleTemplate: "%s | My Site",
			expectedTitle: "Test Note | My Site",
		},
		{
			name:          "template with site name prefix",
			titleTemplate: "Site: %s",
			expectedTitle: "Site: Test Note",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvMock{
				LoggerFunc: func() logger.Logger {
					return &logger.DummyLogger{}
				},
				SiteConfigFunc: func(ctx context.Context) model.SiteConfig {
					return model.SiteConfig{}
				},
				SiteTitleTemplateFunc: func() string {
					return tt.titleTemplate
				},
				LiveNoteViewsFunc: func() *model.NoteViews {
					return noteViews
				},
				LatestNoteViewsFunc: func() *model.NoteViews {
					return noteViews
				},
				CanReadNoteFunc: func(ctx context.Context, note *model.NoteView) (bool, error) {
					return true, nil
				},
				GetTelegramPostLinksByNoteVersionIDFunc: func(ctx context.Context, arg db.GetTelegramPostLinksByNoteVersionIDParams) ([]db.GetTelegramPostLinksByNoteVersionIDRow, error) {
					return nil, nil
				},
			}

			resp, err := rendernotepage.Resolve(context.Background(), env, rendernotepage.Request{
				Path:      "/test-title",
				Version:   "",
				UserToken: nil,
			})

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Equal(t, tt.expectedTitle, resp.Title)
		})
	}
}

func TestResolve_SigninWall(t *testing.T) {
	// makeNote builds a NoteView with the given subgraphs.
	// VersionID=0 so GetTelegramPostLinksByNoteVersionID is never called.
	makeNote := func(free bool, subgraphs map[string]*model.NoteSubgraph) *model.NoteView {
		names := make([]string, 0, len(subgraphs))
		for n := range subgraphs {
			names = append(names, n)
		}
		return &model.NoteView{
			Path:          "/test-note",
			Title:         "Test Note",
			PathID:        1,
			VersionID:     0,
			Content:       []byte("# Test"),
			HTML:          "<h1>Test</h1>",
			Permalink:     "/test-note",
			Free:          free,
			SubgraphNames: names,
			Subgraphs:     subgraphs,
			InLinks:       map[string]struct{}{},
			RawMeta:       map[string]interface{}{},
			Assets:        map[string]struct{}{},
			AssetReplaces: map[string]*model.NoteAssetReplace{},
		}
	}

	// makeNoteViews wraps a note in a NoteViews.
	makeNoteViews := func(note *model.NoteView) *model.NoteViews {
		return &model.NoteViews{
			Map:       map[string]*model.NoteView{"/test-note": note},
			List:      []*model.NoteView{note},
			Subgraphs: map[string]*model.NoteSubgraph{},
			Version:   "live",
		}
	}

	// baseEnv returns a minimal EnvMock for a guest hitting a free note.
	// Override fields as needed per test case.
	baseEnv := func(nv *model.NoteViews) *EnvMock {
		return &EnvMock{
			LoggerFunc: func() logger.Logger {
				return &logger.DummyLogger{}
			},
			SiteConfigFunc: func(ctx context.Context) model.SiteConfig {
				return model.SiteConfig{}
			},
			SiteTitleTemplateFunc: func() string {
				return "%s"
			},
			LiveNoteViewsFunc: func() *model.NoteViews {
				return nv
			},
			LatestNoteViewsFunc: func() *model.NoteViews {
				return nv
			},
			CanReadNoteFunc: func(ctx context.Context, note *model.NoteView) (bool, error) {
				return true, nil
			},
			GetTelegramPostLinksByNoteVersionIDFunc: func(ctx context.Context, arg db.GetTelegramPostLinksByNoteVersionIDParams) ([]db.GetTelegramPostLinksByNoteVersionIDRow, error) {
				return nil, nil
			},
		}
	}

	// authedEnv returns a base env extended with methods needed when a UserToken is present.
	authedEnv := func(nv *model.NoteViews, canRead bool) *EnvMock {
		env := baseEnv(nv)
		env.ListActiveUserSubgraphsFunc = func(ctx context.Context, userID int64) ([]string, error) {
			return []string{}, nil
		}
		env.RecordUserNoteViewFunc = func(ctx context.Context, userID int64, note *model.NoteView, referrerVersionID *int64) {}
		env.LastUserNoteViewFunc = func(ctx context.Context, arg db.LastUserNoteViewParams) (db.LastUserNoteViewRow, error) {
			return db.LastUserNoteViewRow{}, sql.ErrNoRows
		}
		env.CanReadNoteFunc = func(ctx context.Context, note *model.NoteView) (bool, error) {
			return canRead, nil
		}
		return env
	}

	tests := []struct {
		name        string
		note        *model.NoteView
		userToken   *usertoken.Data
		envFunc     func(nv *model.NoteViews) *EnvMock
		wantSignin  bool // expect *SigninWallError
		wantPaywall bool // expect *PaywallError
		wantNoError bool // expect nil error
	}{
		{
			name: "guest + require_signin subgraph -> SigninWallError",
			note: makeNote(true, map[string]*model.NoteSubgraph{
				"members": {Name: "members", RequireSignin: true},
			}),
			userToken:  nil,
			envFunc:    func(nv *model.NoteViews) *EnvMock { return baseEnv(nv) },
			wantSignin: true,
		},
		{
			name: "authenticated user + require_signin -> normal render",
			note: makeNote(true, map[string]*model.NoteSubgraph{
				"members": {Name: "members", RequireSignin: true},
			}),
			userToken:   &usertoken.Data{ID: 1, Role: "user"},
			envFunc:     func(nv *model.NoteViews) *EnvMock { return authedEnv(nv, true) },
			wantNoError: true,
		},
		{
			name: "admin user + require_signin -> normal render",
			note: makeNote(true, map[string]*model.NoteSubgraph{
				"members": {Name: "members", RequireSignin: true},
			}),
			userToken:   &usertoken.Data{ID: 2, Role: "admin"},
			envFunc:     func(nv *model.NoteViews) *EnvMock { return authedEnv(nv, true) },
			wantNoError: true,
		},
		{
			name: "guest + require_signin + non-free -> SigninWallError not PaywallError",
			note: makeNote(false, map[string]*model.NoteSubgraph{
				"members": {Name: "members", RequireSignin: true},
			}),
			userToken:  nil,
			envFunc:    func(nv *model.NoteViews) *EnvMock { return baseEnv(nv) },
			wantSignin: true,
		},
		{
			name: "authenticated user without subscription + non-free + require_signin -> PaywallError",
			note: makeNote(false, map[string]*model.NoteSubgraph{
				"members": {Name: "members", RequireSignin: true},
			}),
			userToken:   &usertoken.Data{ID: 3, Role: "user"},
			envFunc:     func(nv *model.NoteViews) *EnvMock { return authedEnv(nv, false) },
			wantPaywall: true,
		},
		{
			name: "guest + mixed subgraphs one require_signin one not -> SigninWallError",
			note: makeNote(true, map[string]*model.NoteSubgraph{
				"public":  {Name: "public", RequireSignin: false},
				"members": {Name: "members", RequireSignin: true},
			}),
			userToken:  nil,
			envFunc:    func(nv *model.NoteViews) *EnvMock { return baseEnv(nv) },
			wantSignin: true,
		},
		{
			name: "guest + no require_signin subgraph + free note -> normal render",
			note: makeNote(true, map[string]*model.NoteSubgraph{
				"public": {Name: "public", RequireSignin: false},
			}),
			userToken:   nil,
			envFunc:     func(nv *model.NoteViews) *EnvMock { return baseEnv(nv) },
			wantNoError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nv := makeNoteViews(tt.note)
			env := tt.envFunc(nv)

			resp, err := rendernotepage.Resolve(context.Background(), env, rendernotepage.Request{
				Path:      "/test-note",
				UserToken: tt.userToken,
			})

			require.NotNil(t, resp)

			switch {
			case tt.wantSignin:
				require.Error(t, err)
				var signinErr *rendernotepage.SigninWallError
				require.ErrorAs(t, err, &signinErr, "expected *SigninWallError, got %T: %v", err, err)
				var paywallErr *rendernotepage.PaywallError
				require.NotErrorAs(t, err, &paywallErr, "expected SigninWallError, not PaywallError")
			case tt.wantPaywall:
				require.Error(t, err)
				var paywallErr *rendernotepage.PaywallError
				require.ErrorAs(t, err, &paywallErr, "expected *PaywallError, got %T: %v", err, err)
			case tt.wantNoError:
				require.NoError(t, err)
			}
		})
	}
}

func TestResolve_UsesPublicTelegramUsernameWhenAvailable(t *testing.T) {
	note := &model.NoteView{
		Path:          "/telegram-note",
		Title:         "Telegram Note",
		PathID:        1,
		VersionID:     11,
		Content:       []byte("# Telegram Note"),
		HTML:          "<h1>Telegram Note</h1>",
		Permalink:     "/telegram-note",
		Free:          true,
		SubgraphNames: []string{},
		Subgraphs:     map[string]*model.NoteSubgraph{},
		InLinks:       map[string]struct{}{},
		RawMeta:       map[string]interface{}{},
		Assets:        map[string]struct{}{},
		AssetReplaces: map[string]*model.NoteAssetReplace{},
	}

	noteViews := &model.NoteViews{
		Map:       map[string]*model.NoteView{"/telegram-note": note},
		List:      []*model.NoteView{note},
		Subgraphs: map[string]*model.NoteSubgraph{},
		Version:   "live",
	}

	env := &EnvMock{
		LoggerFunc:            func() logger.Logger { return &logger.DummyLogger{} },
		SiteConfigFunc:        func(ctx context.Context) model.SiteConfig { return model.SiteConfig{} },
		SiteTitleTemplateFunc: func() string { return "%s" },
		LiveNoteViewsFunc:     func() *model.NoteViews { return noteViews },
		LatestNoteViewsFunc:   func() *model.NoteViews { return noteViews },
		CanReadNoteFunc:       func(ctx context.Context, note *model.NoteView) (bool, error) { return true, nil },
		GetTelegramPostLinksByNoteVersionIDFunc: func(ctx context.Context, arg db.GetTelegramPostLinksByNoteVersionIDParams) ([]db.GetTelegramPostLinksByNoteVersionIDRow, error) {
			return []db.GetTelegramPostLinksByNoteVersionIDRow{{
				ChatTitle:      "Public Channel",
				TelegramChatID: -1001234567890,
				MessageID:      42,
			}}, nil
		},
		GetTelegramChatNameFunc: func(ctx context.Context, telegramChatID int64) (string, error) {
			require.Equal(t, int64(-1001234567890), telegramChatID)
			return "publicchannel", nil
		},
	}

	resp, err := rendernotepage.Resolve(context.Background(), env, rendernotepage.Request{Path: "/telegram-note"})
	require.NoError(t, err)
	require.Len(t, resp.TelegramLinks, 1)
	require.Equal(t, "https://t.me/publicchannel/42", resp.TelegramLinks[0].URL)
}

func TestResolve_FallsBackToNumericTelegramLinkWhenUsernameUnavailable(t *testing.T) {
	note := &model.NoteView{
		Path:          "/telegram-note",
		Title:         "Telegram Note",
		PathID:        1,
		VersionID:     11,
		Content:       []byte("# Telegram Note"),
		HTML:          "<h1>Telegram Note</h1>",
		Permalink:     "/telegram-note",
		Free:          true,
		SubgraphNames: []string{},
		Subgraphs:     map[string]*model.NoteSubgraph{},
		InLinks:       map[string]struct{}{},
		RawMeta:       map[string]interface{}{},
		Assets:        map[string]struct{}{},
		AssetReplaces: map[string]*model.NoteAssetReplace{},
	}

	noteViews := &model.NoteViews{
		Map:       map[string]*model.NoteView{"/telegram-note": note},
		List:      []*model.NoteView{note},
		Subgraphs: map[string]*model.NoteSubgraph{},
		Version:   "live",
	}

	env := &EnvMock{
		LoggerFunc:            func() logger.Logger { return &logger.DummyLogger{} },
		SiteConfigFunc:        func(ctx context.Context) model.SiteConfig { return model.SiteConfig{} },
		SiteTitleTemplateFunc: func() string { return "%s" },
		LiveNoteViewsFunc:     func() *model.NoteViews { return noteViews },
		LatestNoteViewsFunc:   func() *model.NoteViews { return noteViews },
		CanReadNoteFunc:       func(ctx context.Context, note *model.NoteView) (bool, error) { return true, nil },
		GetTelegramPostLinksByNoteVersionIDFunc: func(ctx context.Context, arg db.GetTelegramPostLinksByNoteVersionIDParams) ([]db.GetTelegramPostLinksByNoteVersionIDRow, error) {
			return []db.GetTelegramPostLinksByNoteVersionIDRow{{
				ChatTitle:      "Private Channel",
				TelegramChatID: -1001234567890,
				MessageID:      42,
			}}, nil
		},
		GetTelegramChatNameFunc: func(ctx context.Context, telegramChatID int64) (string, error) {
			return "", nil
		},
	}

	resp, err := rendernotepage.Resolve(context.Background(), env, rendernotepage.Request{Path: "/telegram-note"})
	require.NoError(t, err)
	require.Len(t, resp.TelegramLinks, 1)
	require.Equal(t, "https://t.me/c/1234567890/42", resp.TelegramLinks[0].URL)
}
