package processnotionwebhook

import (
	"fmt"
	"net/http"
	"trip2g/internal/appreq"
)

type Endpoint struct{}

func (e *Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	env := req.Env.(Env)

	fmt.Println(string(req.Req.PostBody())) //nolint:forbidigo // debug

	id := req.Req.URI().QueryArgs().Peek("id")

	resolveReq := Request{
		ID:   string(id),
		Body: req.Req.PostBody(),
	}

	err := Resolve(req.Req, env, resolveReq)
	if err != nil {
		return nil, err
	}

	ok := map[string]bool{"ok": true}

	return ok, nil
}

func (*Endpoint) Path() string {
	return "/api/notion/webhook"
}

func (*Endpoint) Method() string {
	return http.MethodPost
}
