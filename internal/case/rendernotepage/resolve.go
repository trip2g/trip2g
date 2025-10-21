package rendernotepage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/usertoken"
)

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
	LatestConfig() db.ConfigVersion
	CanReadNote(ctx context.Context, note *model.NoteView) (bool, error)
}

type Request struct {
	Path string

	Version string

	Referrer string

	UserToken *usertoken.Data
}

type VersionBanner struct {
	Label     string
	Permalink string
}

type Response struct {
	Title string
	Note  *model.NoteView
	Notes *model.NoteViews

	NoteSubgraphs []string
	UserSubgraphs []string

	UserToken *usertoken.Data
	UserRole  string
	Time      int

	versionBanner *VersionBanner

	DefaultVersion string

	ViewedAt *time.Time

	IsAdmin bool
}

func (r *Response) NoteSubgraphsJSON() string {
	raw, err := json.Marshal(r.Note.SubgraphNames)
	if err != nil {
		return "null"
	}

	return string(raw)
}

var ErrNotFound = errors.New("page not found")

type PaywallError struct {
	Message string
}

func (e *PaywallError) Error() string {
	return fmt.Sprintf("paywall: %s", e.Message)
}

var systemRE = regexp.MustCompile(`\/_`)

func Resolve(ctx context.Context, env Env, request Request) (*Response, error) {
	var notes *model.NoteViews

	response := Response{
		DefaultVersion: "live",
		UserRole:       "guest",
	}

	// only admins can access the latest version
	isAdmin := request.UserToken.IsAdmin()
	isLatest := request.Version == "latest"

	response.IsAdmin = isAdmin

	config := env.LatestConfig()

	if isAdmin || config.ShowDraftVersions {
		response.DefaultVersion = "latest"

		// admins view the latest version by default
		if request.Version == "" {
			isLatest = true
		}
	} else {
		isLatest = false
	}

	if isLatest {
		notes = env.LatestNoteViews()
	} else {
		notes = env.LiveNoteViews()
	}

	path := request.Path

	if systemRE.MatchString(path) {
		return &response, ErrNotFound
	}

	note := notes.GetByPath(path)
	if note == nil {
		return &response, ErrNotFound
	}

	if isAdmin {
		checkLatestBanner(env, &response, isLatest, path, note)
	}

	// TODO: extract subgraphs
	// TODO: hide all _* pages (system)
	// TODO: add hideSidebar logic

	response.Title = note.Title
	response.Note = note
	response.Notes = notes
	response.UserToken = request.UserToken
	response.Time = int(time.Now().Unix())

	// not sure if this is the right place to do this
	for key := range note.InLinks {
		if len(key) > 1 && key[1] == '_' {
			delete(note.InLinks, key)
		}
	}

	// hide all non-free pages from guests
	if !note.Free && request.UserToken == nil {
		return &response, &PaywallError{Message: "Need auth"}
	}

	if request.UserToken != nil {
		response.UserRole = request.UserToken.Role

		err := handleUserToken(ctx, env, request.UserToken, &response, note, notes, request.Referrer)
		if err != nil {
			return &response, err
		}
	}

	hasAccess, err := env.CanReadNote(ctx, note)
	if err != nil {
		return &response, fmt.Errorf("failed to check note access: %w", err)
	}

	if !hasAccess {
		return &response, &PaywallError{Message: "Need subscription"}
	}

	return &response, nil
}

func checkLatestBanner(env Env, response *Response, isLatest bool, path string, note *model.NoteView) {
	var alternativeNotes *model.NoteViews

	if isLatest {
		alternativeNotes = env.LiveNoteViews()
	} else {
		alternativeNotes = env.LatestNoteViews()
	}

	alternativeNote := alternativeNotes.GetByPath(path)
	if alternativeNote != nil && alternativeNote.VersionID != note.VersionID {
		response.versionBanner = &VersionBanner{
			Permalink: alternativeNotes.ResolveURL(alternativeNote, response.DefaultVersion),
		}

		if isLatest {
			response.versionBanner.Label = "Это последняя загруженная версия, которая отличается от опубликованной"
		} else {
			response.versionBanner.Label = "Это последняя опубликованная версия, которая отличается от загруженной"
		}
	}
}

// handleUserToken processes user token and updates response with user-specific data.
func handleUserToken(
	ctx context.Context,
	env Env,
	userToken *usertoken.Data,
	response *Response,
	note *model.NoteView,
	notes *model.NoteViews,
	referrer string,
) error {
	userID := int64(userToken.ID)

	var err error

	response.UserSubgraphs, err = env.ListActiveUserSubgraphs(ctx, userID)
	if err != nil {
		return err
	}

	referrerVersionID := extractReferrerVersionID(referrer, notes)

	// Record user note view asynchronously to avoid blocking the response
	// and prevent SQLite locking issues during concurrent requests
	go func() {
		// Use a background context to avoid cancellation when request completes
		bgCtx := context.Background()
		env.RecordUserNoteView(bgCtx, userID, note, referrerVersionID)
	}()

	lastViewParams := db.LastUserNoteViewParams{
		UserID: userID,
		PathID: note.PathID,
	}

	lastView, err := env.LastUserNoteView(ctx, lastViewParams)
	if err != nil {
		if !db.IsNoFound(err) {
			return fmt.Errorf("failed to get last user note view: %w", err)
		}
	} else {
		response.ViewedAt = &lastView.CreatedAt
	}

	return nil
}

// extractReferrerVersionID tries to find the version ID from a referrer URL.
func extractReferrerVersionID(referrer string, notes *model.NoteViews) *int64 {
	if referrer == "" {
		return nil
	}

	// Parse the referrer URL
	u, err := url.Parse(referrer)
	if err != nil {
		return nil
	}

	// Extract the path from the referrer
	referrerPath := u.Path
	if referrerPath == "" {
		return nil
	}

	// Clean up path (remove trailing slash, etc.)
	referrerPath = strings.TrimSuffix(referrerPath, "/")
	if referrerPath == "" {
		referrerPath = "/"
	}

	// Try to find the note by path
	referrerNote := notes.GetByPath(referrerPath)
	if referrerNote == nil {
		return nil
	}

	return &referrerNote.VersionID
}
