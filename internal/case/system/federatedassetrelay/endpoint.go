// Package federatedassetrelay streams a federated peer's attachment to whoever
// holds a link the federated_asset tool issued.
//
// The bytes come from the peer and go out as they arrive; nothing is stored
// here, and the federation key that fetches them never leaves this instance.
// The link was issued to a caller who could reach that peer, and until it
// expires it is the whole credential — see federatedassetlink.
package federatedassetrelay

import (
	"context"
	"errors"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strconv"

	"trip2g/internal/appreq"
	"trip2g/internal/federatedassetlink"
	"trip2g/internal/logger"
	"trip2g/internal/model"
)

// Path is where links point. LinkParam carries the signed link.
const (
	Path      = "/_system/mcp/federated_asset"
	LinkParam = "link"
)

type Env interface {
	model.FederationAssetFetcherFactory
	FederatedAssetLinkSecret() string
	Logger() logger.Logger
}

type Endpoint struct{}

func (*Endpoint) Path() string {
	return Path
}

func (*Endpoint) Method() string {
	return http.MethodGet
}

func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	env := req.Env.(Env)
	ctx := req.Req

	link, err := federatedassetlink.Parse(string(ctx.QueryArgs().Peek(LinkParam)), env.FederatedAssetLinkSecret())
	if err != nil {
		ctx.SetStatusCode(http.StatusNotFound)
		ctx.SetBodyString("the link is invalid or has expired; call federated_asset again")
		//nolint:nilerr // intentional: a bad link is a 404 for its holder, not a 500
		return nil, nil
	}

	asset, err := fetch(ctx, env, link)
	if err != nil {
		status := relayStatus(err)
		env.Logger().Warn("federatedassetrelay: peer refused or failed", "kb_id", link.KBID, "status", status, "error", err)
		ctx.SetStatusCode(status)
		ctx.SetBodyString(http.StatusText(status))
		return nil, nil
	}

	fileName := path.Base(link.Name)
	h := &ctx.Response.Header
	h.Set("Cache-Control", "no-store")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Content-Security-Policy", "sandbox")
	h.Set("Content-Disposition", "attachment; filename="+strconv.Quote(fileName)+"; filename*=UTF-8''"+url.PathEscape(fileName))
	ctx.SetContentType(contentType(fileName))
	ctx.SetBodyStream(asset.Body, int(asset.Size))

	return nil, nil
}

func fetch(ctx context.Context, env Env, link federatedassetlink.Link) (*model.FederatedAsset, error) {
	fetcher, err := env.FederationAssetFetcher(ctx, link.KBID)
	if err != nil {
		return nil, err
	}

	params := model.FederatedAssetParams{NotePath: link.NotePath, Name: link.Name}

	return fetcher.FetchAsset(ctx, params)
}

// relayStatus passes on the two answers that are about the attachment itself;
// anything else went wrong between here and the peer.
func relayStatus(err error) int {
	if errors.Is(err, model.ErrFederatedAssetTooLarge) {
		return http.StatusRequestEntityTooLarge
	}

	var httpErr *model.FederationHTTPError
	if errors.As(err, &httpErr) && (httpErr.Status == http.StatusNotFound || httpErr.Status == http.StatusRequestEntityTooLarge) {
		return httpErr.Status
	}

	return http.StatusBadGateway
}

func contentType(fileName string) string {
	t := mime.TypeByExtension(path.Ext(fileName))
	if t == "" {
		return "application/octet-stream"
	}
	return t
}
