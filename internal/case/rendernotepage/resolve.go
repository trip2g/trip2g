package rendernotepage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/usertoken"
)

type Env interface {
	Logger() logger.Logger
	AllNotes() model.NoteViews
	ListActiveSubgraphsByUserID(ctx context.Context, userID int64) ([]string, error)
}

type Request struct {
	Path string

	UserToken *usertoken.Data
}

type Response struct {
	Title string
	Note  *model.NoteView
	Notes model.NoteViews

	UserToken *usertoken.Data
	Time      int
}

const defaultSidebarPath = "/_sidebar"

func (r *Response) Sidebar() *model.NoteView {
	result := r.Notes[defaultSidebarPath]

	sidebarI, sidebarOk := r.Note.RawMeta["sidebar"]
	if sidebarOk {
		switch s := sidebarI.(type) {
		case string:
			result = r.Notes[s]
		case bool:
			if !s {
				return nil
			}
		}
	}

	return result
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

	page, ok := pages[path]
	if !ok {
		return &response, ErrNotFound
	}

	// TODO: extract subgraphs
	// TODO: hide all _* pages (system)
	// TODO: add hideSidebar logic

	response.Title = page.Title
	response.Note = page
	response.Notes = pages
	response.UserToken = request.UserToken
	response.Time = int(time.Now().Unix())

	pageSubgraphs, err := model.NoteViews{"": page}.Subgraphs()
	if err != nil {
		return &response, err
	}

	userSubgraphs := []string{}

	if request.UserToken != nil {
		userSubgraphs, err = env.ListActiveSubgraphsByUserID(ctx, int64(request.UserToken.ID))
		if err != nil {
			return &response, err
		}
	}

	// not sure if this is the right place to do this
	for key := range page.InLinks {
		if len(key) > 1 && key[1] == '_' {
			delete(page.InLinks, key)
		}
	}

	// hide all non-free pages from guests
	if !page.Free && request.UserToken == nil {
		return &response, &PaywallError{Message: "Need auth"}
	}

	hasAccess := len(pageSubgraphs) == 0

	env.Logger().Debug("check access to subgraph", "pageSubgraphs", pageSubgraphs, "userSubgraphs", userSubgraphs)

	// check if the user has access to the subgraph
	for _, ps := range pageSubgraphs {
		for _, us := range userSubgraphs {
			if ps == us {
				hasAccess = true
				break
			}
		}
	}

	if !hasAccess {
		return &response, &PaywallError{Message: "Need subscription"}
	}

	return &response, nil
}
