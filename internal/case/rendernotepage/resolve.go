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
	AllNotes() *model.NoteViews
	ListActiveSubgraphNamesByUserID(ctx context.Context, userID int64) ([]string, error)
	InsertUserNoteView(ctx context.Context, params db.InsertUserNoteViewParams) error
	UpsertUserNoteDailyView(ctx context.Context, params db.UpsertUserNoteDailyViewParams) (int64, error)
	IncreaseUserNoteViewCount(ctx context.Context, userID int64) error
	ListLatestUserNoteViewPathIDS(ctx context.Context, userID int64) ([]int64, error)
}

type Request struct {
	Path string

	UserToken *usertoken.Data
}

type Response struct {
	Title string
	Note  *model.NoteView
	Notes *model.NoteViews

	LatestNotes []*model.NoteView

	NoteSubgraphs []string
	UserSubgraphs []string

	UserToken *usertoken.Data
	Time      int
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
	pages := env.AllNotes()

	path := request.Path
	if path == "/" {
		path = "/index"
	}

	response := Response{}

	note := pages.GetByPath(path)
	if note == nil {
		return &response, ErrNotFound
	}

	// TODO: extract subgraphs
	// TODO: hide all _* pages (system)
	// TODO: add hideSidebar logic

	response.Title = note.Title
	response.Note = note
	response.Notes = pages
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

		latestNoteIDS, err := env.ListLatestUserNoteViewPathIDS(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to list latest user note view path ids: %w", err)
		}

		idMap := pages.IDMap()

		for _, id := range latestNoteIDS {
			note, ok := idMap[id]
			if ok {
				response.LatestNotes = append(response.LatestNotes, note)
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
