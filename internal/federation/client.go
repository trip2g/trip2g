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

func (c *Client) Search(ctx context.Context, params model.MCPSearchParams) (model.FederationResult, error) {
	return c.callTool(ctx, "search", params)
}

func (c *Client) Similar(ctx context.Context, params model.MCPSimilarParams) (model.FederationResult, error) {
	return c.callTool(ctx, "similar", params)
}

func (c *Client) NoteHTML(ctx context.Context, params model.MCPNoteHTMLParams) (model.FederationResult, error) {
	return c.callTool(ctx, "note_html", params)
}

func (c *Client) FederatedSearch(ctx context.Context, params model.MCPSearchParams) (model.FederationResult, error) {
	return c.callTool(ctx, "federated_search", params)
}

func (c *Client) FederatedSimilar(ctx context.Context, params model.MCPSimilarParams) (model.FederationResult, error) {
	return c.callTool(ctx, "federated_similar", params)
}

func (c *Client) FederatedNoteHTML(ctx context.Context, params model.MCPNoteHTMLParams) (model.FederationResult, error) {
	return c.callTool(ctx, "federated_note_html", params)
}

func (c *Client) Expand(ctx context.Context, params model.MCPExpandParams) (model.FederationResult, error) {
	return c.callTool(ctx, "expand", params)
}

func (c *Client) FederatedExpand(ctx context.Context, params model.MCPExpandParams) (model.FederationResult, error) {
	return c.callTool(ctx, "federated_expand", params)
}

func (c *Client) GraphQLRequest(ctx context.Context, params model.MCPGraphQLParams) (model.FederationResult, error) {
	return c.callTool(ctx, "graphql_request", params)
}

func (c *Client) Instructions(ctx context.Context) (model.FederationResult, error) {
	return c.callTool(ctx, "instructions", nil)
}

func (c *Client) FederatedInstructions(ctx context.Context, params model.MCPInstructionsParams) (model.FederationResult, error) {
	return c.callTool(ctx, "federated_instructions", params)
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

	resp, err := c.send(ctx, body, rid, c.peer.Secret)
	if err != nil {
		return model.FederationResult{}, err
	}

	// The peer holds a key this side does not think is current, which is what a
	// rotation looks like from either end until one call has confirmed it. The
	// previous key is the only other one it can be, so this is a single retry,
	// not a loop.
	if resp.Error != nil && resp.Error.Code == model.FederationAuthErrorCode && len(c.peer.PrevSecret) > 0 {
		resp, err = c.send(ctx, body, rid, c.peer.PrevSecret)
		if err != nil {
			return model.FederationResult{}, err
		}
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

// send posts one already-marshalled request signed with the given key. The body
// is signed rather than merely accompanied by a signature, so the same bytes
// have to arrive for the peer to accept them.
func (c *Client) send(ctx context.Context, body []byte, rid string, secret []byte) (rpcResponse, error) {
	headers := map[string]string{
		"X-MCP-Federation-Depth": strconv.Itoa(c.peer.Depth + 1),
	}
	if len(secret) > 0 && c.peer.KID != "" {
		token, err := signOutbound(secret, c.peer.KID, c.peer.Issuer, rid, body)
		if err != nil {
			return rpcResponse{}, fmt.Errorf("sign federation request: %w", err)
		}
		headers["Authorization"] = "Bearer " + token
	}

	raw, err := c.postJSON(ctx, c.peer.KBURL, body, headers, c.timeout)
	if err != nil {
		return rpcResponse{}, err
	}

	var resp rpcResponse
	err = easyjson.Unmarshal(raw, &resp)
	if err != nil {
		return rpcResponse{}, fmt.Errorf("decode federation response: %w", err)
	}
	return resp, nil
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
	// MCP Streamable HTTP requires clients to accept both media types; a peer
	// rejects the request outright without this.
	req.Header.Set("Accept", "application/json, text/event-stream")
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
	// Bh binds the request body to the signature. Without it the JWT authorises
	// a caller and says nothing about what they sent, so anything able to touch
	// the connection can rewrite the arguments inside the 30-second window. That
	// is untidy for a search and fatal for a call that carries a key.
	Bh string `json:"bh,omitempty"`
}

func signOutbound(secret []byte, kid, iss, rid string, body []byte) (string, error) {
	issuedAt := time.Now()
	headerJSON, err := json.Marshal(jwtHeader{Alg: "HS256", Typ: "JWT", KID: kid})
	if err != nil {
		return "", fmt.Errorf("marshal jwt header: %w", err)
	}
	digest := sha256.Sum256(body)
	claimsJSON, err := json.Marshal(jwtClaims{
		Iss: iss,
		Iat: issuedAt.Unix(),
		Exp: issuedAt.Add(30 * time.Second).Unix(),
		Rid: rid,
		Bh:  base64.RawURLEncoding.EncodeToString(digest[:]),
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

// GrantedScope asks the peer which of its subgraphs this pairing may see.
func (c *Client) GrantedScope(ctx context.Context) (model.FederationResult, error) {
	return c.callTool(ctx, model.GrantedScopeTool, nil)
}

// RotateSecret asks the peer to replace this pairing's shared key with the one
// carried here. It signs like every other call — current key first, previous on
// an auth refusal — which is what makes a retry after a lost response reach a
// peer that already applied the change.
func (c *Client) RotateSecret(ctx context.Context, params model.MCPRotateSecretParams) (model.FederationResult, error) {
	return c.callTool(ctx, model.RotateSecretTool, params)
}
