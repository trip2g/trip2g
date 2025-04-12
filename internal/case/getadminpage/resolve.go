package getadminpage

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"trip2g/internal/appreq"
	"trip2g/internal/mdloader"
)

//go:generate easyjson -snake_case -all -no_std_marshalers ./resolve.go

type Env interface {
	PageByPath(path string) (*mdloader.Page, error)
}

type Request struct {
	Path string
}

type Response struct {
	Page *mdloader.Page
}

func Resolve(_ context.Context, env Env, req Request) (*Response, error) {
	page, err := env.PageByPath(req.Path)
	if err != nil {
		if errors.Is(err, errors.New("page not found")) {
			return nil, &appreq.Error{
				Code:    http.StatusNotFound,
				Message: "page not found",
			}
		}
		return nil, fmt.Errorf("failed to get page: %w", err)
	}

	return &Response{
		Page: page,
	}, nil
}
