package rendernotepage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/url"
	"regexp"
	"strings"
	"time"

	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/templateviews"
	"trip2g/internal/usertoken"
)

type Env interface {
	Layouts() *model.Layouts

	Logger() logger.Logger
	PublicURL() string
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
}

type Request struct {
	Path string

	Version string

	Referrer string

	Host string

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

	NoteView *templateviews.Note

	// add NoteView...

	NoteSubgraphs []string
	UserSubgraphs []string

	UserToken *usertoken.Data
	UserRole  string
	Time      int

	versionBanner *VersionBanner

	DefaultVersion string

	ViewedAt *time.Time

	Config struct {
		ShowDraftVersions bool
		DefaultLayout     string
	}

	IsAdmin        bool
	OnboardingMode bool

	// domainHost is the normalized custom domain host for this request.
	// Empty string for main domain requests.
	// Used by NoteHTML() to select domain-specific rendered HTML.
	domainHost string
}

func (r *Response) NoteSubgraphsJSON() string {
	raw, err := json.Marshal(r.Note.SubgraphNames)
	if err != nil {
		return "null"
	}

	return string(raw)
}

// NoteHTML returns the note's HTML, using domain-specific pre-rendered HTML if available.
// For custom domain requests, domainHost is the normalized host (e.g. "foo.com").
// For main domain requests, domainHost is "" — DomainHTML[""] holds main-domain re-rendered
// HTML where links to custom-domain-only notes use full URLs (https://foo.com/path).
func (r *Response) NoteHTML() template.HTML {
	if r.Note == nil {
		return ""
	}
	if r.Note.DomainHTML != nil {
		if domainHTML, ok := r.Note.DomainHTML[r.domainHost]; ok {
			return domainHTML
		}
	}
	return r.Note.HTML
}

// SidebarHTML returns domain-aware HTML for a sidebar note.
func (r *Response) SidebarHTML(sidebar *model.NoteView) template.HTML {
	if sidebar == nil {
		return ""
	}
	if sidebar.DomainHTML != nil {
		if domainHTML, ok := sidebar.DomainHTML[r.domainHost]; ok {
			return domainHTML
		}
	}
	return sidebar.HTML
}

var ErrNotFound = errors.New("page not found")

const (
	versionLive   = "live"
	versionLatest = "latest"
)

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
		DefaultVersion: versionLive,
		UserRole:       "guest",
	}

	isAdmin := request.UserToken.IsAdmin()

	response.IsAdmin = isAdmin

	siteConfig := env.SiteConfig(ctx)
	response.Config.ShowDraftVersions = siteConfig.ShowDraftVersions
	response.Config.DefaultLayout = siteConfig.DefaultLayout

	// Default: everyone sees live (or latest if ShowDraftVersions is on).
	// Admin can explicitly switch via ?version=latest or ?version=live.
	isLatest := siteConfig.ShowDraftVersions

	if isAdmin && request.Version != "" {
		isLatest = request.Version == versionLatest
	}

	if siteConfig.ShowDraftVersions {
		response.DefaultVersion = versionLatest
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

	// Show onboarding page if there are no notes at all.
	if len(env.LatestNoteViews().Map) == 0 {
		response.OnboardingMode = true
		return &response, nil
	}

	// Only resolve publicURL when host is present (avoids unnecessary call for main-domain requests).
	var publicURL string
	if request.Host != "" {
		publicURL = env.PublicURL()
	}
	note := resolveNote(notes, request.Host, path, publicURL)
	if note == nil {
		return &response, ErrNotFound
	}

	// Set domain context for domain-aware HTML rendering.
	normalizedHost := model.NormalizeDomain(request.Host)
	mainHost := model.NormalizeDomain(model.ExtractHost(publicURL))
	if normalizedHost != mainHost && normalizedHost != "" && notes.IsCustomDomain(normalizedHost) {
		response.domainHost = normalizedHost
	}

	if isAdmin {
		checkLatestBanner(env, &response, isLatest, path, note)
	}

	// TODO: extract subgraphs
	// TODO: hide all _* pages (system)
	// TODO: add hideSidebar logic

	response.Title = formatTitle(note.Title, env.SiteTitleTemplate())
	response.Note = note
	response.Notes = notes
	response.UserToken = request.UserToken
	response.Time = int(time.Now().Unix())
	response.NoteView = templateviews.NewNoteWithDomain(note, response.domainHost)

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
	var alternativeVersion string

	if isLatest {
		alternativeNotes = env.LiveNoteViews()
		alternativeVersion = versionLive
	} else {
		alternativeNotes = env.LatestNoteViews()
		alternativeVersion = versionLatest
	}

	alternativeNote := alternativeNotes.GetByPath(path)
	if alternativeNote != nil && alternativeNote.VersionID != note.VersionID {
		// Build permalink with version parameter for switching.
		permalink := alternativeNote.Permalink
		u, err := url.Parse(permalink)
		if err == nil {
			query := u.Query()
			query.Set("version", alternativeVersion)
			u.RawQuery = query.Encode()
			permalink = u.String()
		}

		response.versionBanner = &VersionBanner{
			Permalink: permalink,
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

// formatTitle applies the site title template to the note title.
// Template must contain %s which will be replaced with the note title.
func formatTitle(noteTitle, template string) string {
	return fmt.Sprintf(template, noteTitle)
}

// resolveNote looks up a note using domain-aware routing.
//
// Known custom domain: only notes with an explicit route for that domain are served.
// No fallback to the global permalink map — isolation is strict.
//
// Main domain or unknown host: checks RouteMap[""] alias routes first, then nv.Map
// (permalink-based lookup). Unknown hosts (e.g. localhost in dev/tests) are treated
// as the main domain because they have no entries in RouteMap.
func resolveNote(notes *model.NoteViews, host, path, publicURL string) *model.NoteView {
	normalizedHost := model.NormalizeDomain(host)
	mainHost := model.NormalizeDomain(model.ExtractHost(publicURL))

	isKnownCustomDomain := normalizedHost != mainHost && normalizedHost != "" && notes.IsCustomDomain(normalizedHost)

	if isKnownCustomDomain {
		// Known custom domain: only serve notes with an explicit route for this domain.
		// No fallthrough to nv.Map — notes without an explicit route return nil (404).
		return notes.GetByRoute(normalizedHost, path)
	}

	// Main domain or unknown host: check RouteMap[""] first (alias routes), then nv.Map.
	if note := notes.GetByRoute("", path); note != nil {
		return note
	}

	// Fallback: permalink-based lookup (main domain only).
	return notes.GetByPath(path)
}
