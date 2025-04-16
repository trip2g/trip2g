package rendernotepage

import (
	"context"
	"errors"

	"trip2g/internal/mdloader"
	"trip2g/internal/usertoken"
)

type Env interface {
	AllPages() map[string]*mdloader.Page
}

type Request struct {
	Path string

	UserToken *usertoken.Data
}

type Response struct {
	Title string
	Page  *mdloader.Page
	Pages map[string]*mdloader.Page
}

var ErrNotFound = errors.New("page not found")
var ErrPaywall = errors.New("paywall")

func Resolve(ctx context.Context, env Env, request Request) (*Response, error) {
	pages := env.AllPages()

	path := request.Path
	if path == "/" {
		path = "/index"
	}

	page, ok := pages[path]
	if !ok {
		return nil, ErrNotFound
	}

	response := Response{
		Title: page.Title,
		Page:  page,
		Pages: pages,
	}

	if !page.Free && request.UserToken == nil {
		return &response, ErrPaywall
	}

	return &response, nil
}
