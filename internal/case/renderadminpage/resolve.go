package renderadminpage

import (
	"context"
	"trip2g/internal/model"
)

type Env interface {
	AdminJSURL() string
	EditorJSURL() string
	EditorLocaleHashes() map[string]string
	LiveNoteViews() *model.NoteViews
}

type Request struct{}

type Response struct {
	JSURL              string
	EditorJSURL        string
	EditorLocaleHashes map[string]string
}

func Resolve(ctx context.Context, env Env, request Request) (*Response, error) {
	return &Response{
		JSURL:              env.AdminJSURL(),
		EditorJSURL:        env.EditorJSURL(),
		EditorLocaleHashes: env.EditorLocaleHashes(),
	}, nil
}
