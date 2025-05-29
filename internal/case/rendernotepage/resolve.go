package rendernotepage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	Logger() logger.Logger
	LatestNoteViews() *model.NoteViews
	LiveNoteViews() *model.NoteViews
	ListActiveSubgraphNamesByUserID(ctx context.Context, userID int64) ([]string, error)
	InsertUserNoteView(ctx context.Context, params db.InsertUserNoteViewParams) error
	UpsertUserNoteDailyView(ctx context.Context, params db.UpsertUserNoteDailyViewParams) (int64, error)
	IncreaseUserNoteViewCount(ctx context.Context, userID int64) error
}

type Request struct {
	Path string

	Version string

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
	Time      int

	versionBanner *VersionBanner
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

func Resolve(ctx context.Context, env Env, request Request) (*Response, error) {
	var notes *model.NoteViews

	// only admins can access the latest version
	isAdmin := request.UserToken.IsAdmin()
	isLatest := request.Version == "latest" && isAdmin

	if isLatest {
		notes = env.LatestNoteViews()
	} else {
		notes = env.LiveNoteViews()
	}

	path := request.Path
	if path == "/" {
		path = "/index"
	}

	response := Response{}

	note := notes.GetByPath(path)
	if note == nil {
		return &response, ErrNotFound
	}

	if isAdmin {
		var alternativeNotes *model.NoteViews

		if isLatest {
			alternativeNotes = env.LiveNoteViews()
		} else {
			alternativeNotes = env.LatestNoteViews()
		}

		alternativeNote := alternativeNotes.GetByPath(path)
		if alternativeNote != nil && alternativeNote.VersionID != note.VersionID {
			response.versionBanner = &VersionBanner{
				Permalink: alternativeNotes.ResolveURL(alternativeNote),
			}

			if isLatest {
				response.versionBanner.Label = "Это последняя загруженная версия, которая отличается от опубликованной"
			} else {
				response.versionBanner.Label = "Это последняя опубликованная версия, которая отличается от загруженной"
			}
		}
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
		userID := int64(request.UserToken.ID)

		var err error

		response.UserSubgraphs, err = env.ListActiveSubgraphNamesByUserID(ctx, userID)
		if err != nil {
			return &response, err
		}

		const maxCount = int64(100)

		dailyParams := db.UpsertUserNoteDailyViewParams{
			UserID: userID,
			PathID: note.PathID,
		}

		dailyCount, err := env.UpsertUserNoteDailyView(ctx, dailyParams)
		if err != nil {
			return nil, fmt.Errorf("failed to upsert user note daily view: %w", err)
		}

		// TODO: read from the app config
		if dailyCount < maxCount {
			err = env.InsertUserNoteView(ctx, db.InsertUserNoteViewParams(dailyParams))
			if err != nil {
				return nil, fmt.Errorf("failed to insert user note view: %w", err)
			}

			err = env.IncreaseUserNoteViewCount(ctx, userID)
			if err != nil {
				return nil, fmt.Errorf("failed to increase user note view count: %w", err)
			}
		}
	}

	hasAccess := len(response.Note.Subgraphs) == 0

	if request.UserToken != nil && request.UserToken.Role == "admin" {
		hasAccess = true
	}

	// check if the user has access to the subgraph
	if !hasAccess {
		for _, ps := range response.Note.SubgraphNames {
			for _, us := range response.UserSubgraphs {
				if ps == us {
					hasAccess = true
					break
				}
			}
		}
	}

	if !hasAccess {
		return &response, &PaywallError{Message: "Need subscription"}
	}

	return &response, nil
}
