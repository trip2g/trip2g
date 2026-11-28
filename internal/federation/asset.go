package federation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"trip2g/internal/model"

	"github.com/valyala/fasthttp"
)

const maxAssetDescriptionSize = 64 << 10

var ErrAssetMisdescribed = errors.New("the peer described an attachment other than the one asked for")

// AssetClient asks a peer about one of its notes' attachments and streams it.
//
// A sibling of Client rather than more methods on it: the tool calls are small
// JSON answers under one short deadline, and a file is neither. It needs a
// connection that streams and a deadline sized for a transfer.
type AssetClient struct {
	peer   model.FederationPeer
	http   *fasthttp.Client
	limits AssetLimits
}

// AssetLimits bounds one transfer: MaxSize is the largest body accepted, and
// Timeout covers all of it, from the request to the last byte.
type AssetLimits struct {
	MaxSize int64
	Timeout time.Duration
}

// NewAssetClient builds an asset client for one peer. http must stream
// response bodies (StreamResponseBody).
func NewAssetClient(peer model.FederationPeer, http *fasthttp.Client, limits AssetLimits) *AssetClient {
	return &AssetClient{peer: peer, http: http, limits: limits}
}

func (c *AssetClient) DescribeAsset(ctx context.Context, params model.FederatedAssetParams) (model.NoteAssetInfo, error) {
	params.Describe = true

	resp, err := c.request(ctx, params)
	if err != nil {
		return model.NoteAssetInfo{}, err
	}
	defer releaseStreamed(resp)

	raw, err := io.ReadAll(io.LimitReader(resp.BodyStream(), maxAssetDescriptionSize))
	if err != nil {
		return model.NoteAssetInfo{}, fmt.Errorf("read federated asset description: %w", err)
	}

	var info model.NoteAssetInfo
	err = json.Unmarshal(raw, &info)
	if err != nil {
		return model.NoteAssetInfo{}, fmt.Errorf("decode federated asset description: %w", err)
	}
	if !describesAsked(info, params) {
		return model.NoteAssetInfo{}, ErrAssetMisdescribed
	}
	if info.Size > c.limits.MaxSize {
		return model.NoteAssetInfo{}, model.ErrFederatedAssetTooLarge
	}

	return info, nil
}

func describesAsked(info model.NoteAssetInfo, params model.FederatedAssetParams) bool {
	if info.Name != params.Name || info.Size < 0 || len(info.Sha256) != sha256.Size*2 {
		return false
	}

	digest, err := hex.DecodeString(info.Sha256)
	return err == nil && hex.EncodeToString(digest) == info.Sha256
}

func (c *AssetClient) FetchAsset(ctx context.Context, params model.FederatedAssetParams) (*model.FederatedAsset, error) {
	params.Describe = false

	resp, err := c.request(ctx, params)
	if err != nil {
		return nil, err
	}

	size := int64(resp.Header.ContentLength())
	if size > c.limits.MaxSize {
		releaseStreamed(resp)
		return nil, model.ErrFederatedAssetTooLarge
	}
	if size < 0 {
		body := &cappedBody{ReadCloser: &streamedBody{resp: resp}, remaining: c.limits.MaxSize}
		return &model.FederatedAsset{Size: -1, Body: body}, nil
	}

	return &model.FederatedAsset{Size: size, Body: &streamedBody{resp: resp}}, nil
}

// cappedBody holds a body sent without Content-Length to the size limit as it
// is read: the byte past the limit is never handed on, and the read that meets
// it fails with ErrFederatedAssetTooLarge.
type cappedBody struct {
	io.ReadCloser
	remaining int64
}

func (b *cappedBody) Read(p []byte) (int, error) {
	if b.remaining < 0 {
		return 0, model.ErrFederatedAssetTooLarge
	}
	if int64(len(p)) > b.remaining+1 {
		p = p[:b.remaining+1]
	}

	n, err := b.ReadCloser.Read(p)
	b.remaining -= int64(n)
	if b.remaining < 0 {
		return n - 1, model.ErrFederatedAssetTooLarge
	}

	return n, err
}

// request sends one signed asset request and returns the response with its
// body still on the wire. It signs like every other federated call: current key
// first, then — on an authentication refusal — the one this side rotated away
// from, which the peer may still be holding.
func (c *AssetClient) request(ctx context.Context, params model.FederatedAssetParams) (*fasthttp.Response, error) {
	if c.peer.KBURL == "" {
		return nil, errors.New("federation peer url is empty")
	}

	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshal federated asset request: %w", err)
	}

	rid := strconv.FormatInt(time.Now().UnixNano(), 10)

	resp, err := c.send(ctx, body, rid, c.peer.Secret)

	var httpErr *model.FederationHTTPError
	if errors.As(err, &httpErr) && httpErr.Status == http.StatusUnauthorized && len(c.peer.PrevSecret) > 0 {
		resp, err = c.send(ctx, body, rid, c.peer.PrevSecret)
	}

	return resp, err
}

func (c *AssetClient) send(ctx context.Context, body []byte, rid string, secret []byte) (*fasthttp.Response, error) {
	err := ctx.Err()
	if err != nil {
		return nil, err
	}

	token, err := signOutbound(secret, c.peer.KID, c.peer.Issuer, rid, body)
	if err != nil {
		return nil, fmt.Errorf("sign federation request: %w", err)
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(c.peer.KBURL + model.FederatedAssetPath)
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetContentType("application/json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Authorization", "Bearer "+token)
	req.SetBody(body)
	// A body abandoned halfway would leave the rest of the file on a pooled
	// connection, where the next response on it would be read from file bytes.
	req.SetConnectionClose()

	resp := fasthttp.AcquireResponse()

	err = c.http.DoTimeout(req, resp, c.limits.Timeout)
	if err != nil {
		fasthttp.ReleaseResponse(resp)
		return nil, fmt.Errorf("federated asset request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		status := resp.StatusCode()
		releaseStreamed(resp)
		return nil, &model.FederationHTTPError{Status: status}
	}

	return resp, nil
}

// streamedBody hands the response back to the pool only once its body has been
// read or abandoned: until then the connection belongs to it.
type streamedBody struct {
	resp *fasthttp.Response
}

func (b *streamedBody) Read(p []byte) (int, error) {
	return b.resp.BodyStream().Read(p)
}

func (b *streamedBody) Close() error {
	err := b.resp.CloseBodyStream()
	fasthttp.ReleaseResponse(b.resp)
	return err
}

func releaseStreamed(resp *fasthttp.Response) {
	_ = resp.CloseBodyStream()
	fasthttp.ReleaseResponse(resp)
}
