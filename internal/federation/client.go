package federation

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"trip2g/internal/model"
	"trip2g/internal/ssrfsafe"

	"github.com/mailru/easyjson"
	"github.com/valyala/fasthttp"
)

const (
	defaultTimeout         = 2 * time.Second
	defaultMaxResponseBody = 1 << 20
	userAgent              = "trip2g-federation/1.0"
)

type Client struct {
	peer    model.FederationPeer
	http    *fasthttp.Client
	timeout time.Duration
}

func NewClient(peer model.FederationPeer, http *fasthttp.Client, devMode bool) *Client {
	if http == nil {
		http = &fasthttp.Client{
			MaxResponseBodySize: defaultMaxResponseBody,
		}
		if !devMode {
			http.DialTimeout = ssrfsafe.DialTimeout
		}
	}
	return &Client{
		peer:    peer,
		http:    http,
		timeout: defaultTimeout,
	}
}

func (c *Client) Search(ctx context.Context, params model.FederationSearchParams) (model.FederationResult, error) {
	return c.callTool(ctx, "search", params)
}

func (c *Client) Similar(ctx context.Context, params model.FederationSimilarParams) (model.FederationResult, error) {
	return c.callTool(ctx, "similar", params)
}

func (c *Client) NoteHTML(ctx context.Context, params model.FederationNoteHTMLParams) (model.FederationResult, error) {
	return c.callTool(ctx, "note_html", params)
}

func (c *Client) FederatedSearch(ctx context.Context, params model.FederationSearchParams) (model.FederationResult, error) {
	return c.callTool(ctx, "federated_search", params)
}

func (c *Client) FederatedSimilar(ctx context.Context, params model.FederationSimilarParams) (model.FederationResult, error) {
	return c.callTool(ctx, "federated_similar", params)
}

func (c *Client) FederatedNoteHTML(ctx context.Context, params model.FederationNoteHTMLParams) (model.FederationResult, error) {
	return c.callTool(ctx, "federated_note_html", params)
}

func (c *Client) Expand(ctx context.Context, params model.FederationExpandParams) (model.FederationResult, error) {
	return c.callTool(ctx, "expand", params)
}

func (c *Client) FederatedExpand(ctx context.Context, params model.FederationExpandParams) (model.FederationResult, error) {
	return c.callTool(ctx, "federated_expand", params)
}

func (c *Client) GraphQLRequest(ctx context.Context, params model.FederationGraphQLParams) (model.FederationResult, error) {
	return c.callTool(ctx, "graphql_request", params)
}

func (c *Client) callTool(ctx context.Context, name string, args any) (model.FederationResult, error) {
	if c == nil {
		return model.FederationResult{}, errors.New("federation client is nil")
	}
	if c.peer.KBURL == "" {
		return model.FederationResult{}, errors.New("federation peer url is empty")
	}

	rid := strconv.FormatInt(time.Now().UnixNano(), 10)
	body, err := easyjson.Marshal(rpcRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params: toolParams{
			Name:      name,
			Arguments: args,
		},
		ID: rid,
	})
	if err != nil {
		return model.FederationResult{}, fmt.Errorf("marshal federation request: %w", err)
	}

	headers := map[string]string{
		"X-MCP-Federation-Depth": strconv.Itoa(c.peer.Depth + 1),
	}
	if len(c.peer.Secret) > 0 && c.peer.KID != "" {
		token, signErr := signOutbound(c.peer.Secret, c.peer.KID, c.peer.Issuer, rid)
		if signErr != nil {
			return model.FederationResult{}, fmt.Errorf("sign federation request: %w", signErr)
		}
		headers["Authorization"] = "Bearer " + token
	}

	raw, err := c.postJSON(ctx, c.peer.KBURL, body, headers, c.timeout)
	if err != nil {
		return model.FederationResult{}, err
	}

	var resp rpcResponse
	if e := easyjson.Unmarshal(raw, &resp); e != nil {
		return model.FederationResult{}, fmt.Errorf("decode federation response: %w", e)
	}
	if resp.Error != nil {
		return model.FederationResult{}, fmt.Errorf("federation rpc error %d: %s", resp.Error.Code, resp.Error.Message)
	}
	content := make([]model.FederationContent, len(resp.Result.Content))
	for i, c := range resp.Result.Content {
		content[i] = model.FederationContent{Type: c.Type, Text: c.Text}
	}
	return model.FederationResult{
		Content:           content,
		StructuredContent: resp.Result.StructuredContent,
		IsError:           resp.Result.IsError,
	}, nil
}

func (c *Client) postJSON(ctx context.Context, url string, body []byte, headers map[string]string, timeout time.Duration) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(url)
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetContentType("application/json")
	req.Header.Set("User-Agent", userAgent)
	req.SetBody(body)

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	httpClient := c.http
	if httpClient == nil {
		httpClient = &fasthttp.Client{MaxResponseBodySize: defaultMaxResponseBody}
	}
	if err := httpClient.DoTimeout(req, resp, timeout); err != nil {
		return nil, fmt.Errorf("post json: %w", err)
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return nil, fmt.Errorf("post json: status %d", resp.StatusCode())
	}

	return append([]byte(nil), resp.Body()...), nil
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
	KID string `json:"kid"`
}

type jwtClaims struct {
	Iss string `json:"iss"`
	Iat int64  `json:"iat"`
	Exp int64  `json:"exp"`
	Rid string `json:"rid"`
}

func signOutbound(secret []byte, kid, iss, rid string) (string, error) {
	issuedAt := time.Now()
	headerJSON, err := json.Marshal(jwtHeader{Alg: "HS256", Typ: "JWT", KID: kid})
	if err != nil {
		return "", fmt.Errorf("marshal jwt header: %w", err)
	}
	claimsJSON, err := json.Marshal(jwtClaims{
		Iss: iss,
		Iat: issuedAt.Unix(),
		Exp: issuedAt.Add(30 * time.Second).Unix(),
		Rid: rid,
	})
	if err != nil {
		return "", fmt.Errorf("marshal jwt claims: %w", err)
	}

	unsigned := base64.RawURLEncoding.EncodeToString(headerJSON) + "." +
		base64.RawURLEncoding.EncodeToString(claimsJSON)
	signature := hmacSHA256(secret, []byte(unsigned))
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func hmacSHA256(secret, payload []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	return mac.Sum(nil)
}
