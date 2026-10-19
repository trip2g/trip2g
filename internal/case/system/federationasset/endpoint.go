package federationasset

import (
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"trip2g/internal/appreq"
	"trip2g/internal/case/mcp"
	"trip2g/internal/model"
)

type Endpoint struct{}

func (*Endpoint) Path() string {
	return "/_system/mcp" + model.FederatedAssetPath
}

func (*Endpoint) Method() string {
	return http.MethodPost
}

func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	env := req.Env.(Env)
	ctx := req.Req
	body := ctx.Request.Body()

	caller, ok := authenticate(req, env, body)
	if !ok {
		ctx.SetStatusCode(http.StatusUnauthorized)
		return nil, nil
	}

	var params model.FederatedAssetParams
	err := json.Unmarshal(body, &params)
	if err != nil || params.NotePath == "" || params.Name == "" {
		ctx.SetStatusCode(http.StatusBadRequest)
		ctx.SetBodyString("note_path and name are required")
		//nolint:nilerr // intentional: a malformed request is a 400, not a 500
		return nil, nil
	}

	asset, err := Resolve(ctx, env, caller, params)
	switch {
	case errors.Is(err, ErrAssetNotFound):
		ctx.SetStatusCode(http.StatusNotFound)
		ctx.SetBodyString(err.Error())
		return nil, nil
	case errors.Is(err, ErrAssetTooLarge):
		ctx.SetStatusCode(http.StatusRequestEntityTooLarge)
		ctx.SetBodyString(err.Error())
		return nil, nil
	case err != nil:
		return nil, err
	}

	if params.Describe {
		info := model.NoteAssetInfo{Name: params.Name, Size: asset.Size, Sha256: asset.Sha256Hash}
		raw, marshalErr := json.Marshal(info)
		if marshalErr != nil {
			return nil, marshalErr
		}
		ctx.SetContentType("application/json")
		ctx.SetBody(raw)
		return nil, nil
	}

	stream, err := env.StreamAssetObject(ctx, asset, 0, asset.Size)
	if err != nil {
		return nil, err
	}

	ctx.SetContentType(contentType(asset.FileName))
	ctx.Response.Header.Set("X-Content-Type-Options", "nosniff")
	ctx.SetBodyStream(stream, int(asset.Size))

	return nil, nil
}

// authenticate accepts only a bearer whose signature covers the body. The body
// is what names the attachment, so a token that binds nothing — the kind a GET
// beside this endpoint carries — would let whoever saw one ask for any
// attachment the pairing can read for as long as it lives.
func authenticate(req *appreq.Request, env Env, body []byte) (mcp.FederationCaller, bool) {
	header := strings.TrimSpace(string(req.Req.Request.Header.Peek("Authorization")))
	token, isBearer := strings.CutPrefix(header, "Bearer ")
	token = strings.TrimSpace(token)
	if !isBearer || token == "" {
		return mcp.FederationCaller{}, false
	}

	caller, err := mcp.VerifyFederationBearer(req.Req, env, token, body)
	if err != nil || !caller.BodyBound {
		return mcp.FederationCaller{}, false
	}

	return caller, true
}

func contentType(fileName string) string {
	t := mime.TypeByExtension(filepath.Ext(fileName))
	if t == "" {
		return "application/octet-stream"
	}
	return t
}
