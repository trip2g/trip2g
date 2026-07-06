package renderadminpage

import (
	"context"
	"trip2g/internal/model"
)

type Env interface {
	AdminJSURL() string
	LiveNoteViews() *model.NoteViews
}

type Request struct{}

type Response struct {
	JSURL string
}

func Resolve(ctx context.Context, env Env, request Request) (*Response, error) {
	return &Response{JSURL: env.AdminJSURL()}, nil
}
