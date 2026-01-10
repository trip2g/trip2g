package renderlayoutpreview

import (
	"encoding/json"
	"net/http"

	"trip2g/internal/appreq"
	"trip2g/internal/layoutloader"
)

type Endpoint struct{}

type requestBody struct {
	NotePath string                  `json:"note_path"`
	Layout   layoutloader.JSONLayout `json:"layout"`
}

type responseBody struct {
	HTML  string `json:"html,omitempty"`
	Error string `json:"error,omitempty"`
}

func (e Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	ctx := req.Req

	// Get user token
	token, err := req.UserToken()
	if err != nil {
		return writeJSON(ctx, http.StatusUnauthorized, responseBody{Error: "unauthorized"})
	}

	// Parse request body
	var body requestBody
	err = json.Unmarshal(ctx.PostBody(), &body)
	if err != nil {
		return writeJSON(ctx, http.StatusBadRequest, responseBody{Error: "invalid JSON: " + err.Error()})
	}

	if body.NotePath == "" {
		return writeJSON(ctx, http.StatusBadRequest, responseBody{Error: "note_path is required"})
	}

	request := Request{
		UserToken: token,
		NotePath:  body.NotePath,
		Layout:    body.Layout,
	}

	resp, err := Resolve(ctx, req.Env.(Env), request)
	if err != nil {
		return writeJSON(ctx, http.StatusBadRequest, responseBody{Error: err.Error()})
	}

	return writeJSON(ctx, http.StatusOK, responseBody{HTML: resp.HTML})
}

func writeJSON(ctx interface{ SetContentType(string); SetStatusCode(int); SetBody([]byte) }, status int, body responseBody) (interface{}, error) {
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(status)
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	ctx.SetBody(data)
	return nil, nil
}

func (Endpoint) Path() string {
	return "/_system/layouts/render"
}

func (Endpoint) Method() string {
	return http.MethodPost
}
