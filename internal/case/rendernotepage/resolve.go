package rendernotepage

import (
	"context"
	"errors"
	"fmt"

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

	fmt.Println("resolve path", path)
	fmt.Printf("pages: %+v\n", pages)

	page, ok := pages[path]
	if !ok {
		return nil, ErrNotFound
	}

	if !page.Free && request.UserToken == nil {
		return nil, ErrPaywall
	}

	response := Response{
		Title: page.Title,
		Page:  page,
		Pages: pages,
	}

	return &response, nil
}
