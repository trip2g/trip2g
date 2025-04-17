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

const defaultSidebarPath = "/_sidebar"

func (r *Response) Sidebar() *mdloader.Page {
	result := r.Pages[defaultSidebarPath]

	sidebarI, sidebarOk := r.Page.RawMeta["sidebar"]
	if sidebarOk {
		switch s := sidebarI.(type) {
		case string:
			result = r.Pages[s]
		case bool:
			if !s {
				return nil
			}
		}
	}

	return result
}

var ErrNotFound = errors.New("page not found")
var ErrPaywall = errors.New("paywall")

func Resolve(ctx context.Context, env Env, request Request) (*Response, error) {
	pages := env.AllPages()

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
	response.Page = page
	response.Pages = pages

	// not sure if this is the right place to do this
	for key := range page.InLinks {
		if len(key) > 1 && key[1] == '_' {
			delete(page.InLinks, key)
		}
	}

	if !page.Free && request.UserToken == nil {
		return &response, ErrPaywall
	}

	return &response, nil
}
