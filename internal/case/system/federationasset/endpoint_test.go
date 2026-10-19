package federationasset_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"

	"trip2g/internal/appreq"
	"trip2g/internal/case/mcp"
	"trip2g/internal/case/system/federationasset"
	"trip2g/internal/db"
	"trip2g/internal/federation"
	"trip2g/internal/logger"
	"trip2g/internal/model"
)

const (
	maxSize   = 1 << 20
	kid       = "consumer"
	secretKey = "0123456789abcdef0123456789abcdef"
	peerURL   = "http://peer.test/_system/mcp"
	sharedZip = "zip bytes the shared note links to, long enough to be streamed in pieces"
)

type peer struct {
	env    *EnvMock
	client *fasthttp.Client
	served map[int64]string
}

// newPeer runs the endpoint behind a real HTTP server, so the bytes cross the
// same streaming path production uses. The pairing may read subgraph "shared"
// only.
func newPeer(t *testing.T) *peer {
	t.Helper()

	shared := &model.NoteView{
		Path:          "skills/reel.md",
		SubgraphNames: []string{"shared"},
		AssetReplaces: map[string]*model.NoteAssetReplace{
			"files/attach.zip": {ID: 1, Hash: "aaa", FileName: "attach.zip"},
			"big.bin":          {ID: 3, Hash: "ccc", FileName: "big.bin"},
		},
	}
	other := &model.NoteView{
		Path:          "private/plan.md",
		SubgraphNames: []string{"other"},
		AssetReplaces: map[string]*model.NoteAssetReplace{
			"plan.pdf": {ID: 2, Hash: "bbb", FileName: "plan.pdf"},
		},
	}
	nvs := model.NewNoteViews()
	nvs.PathMap[shared.Path] = shared
	nvs.PathMap[other.Path] = other

	rows := map[int64]db.NoteAsset{
		1: {ID: 1, FileName: "attach.zip", Sha256Hash: "aaa", Size: int64(len(sharedZip))},
		2: {ID: 2, FileName: "plan.pdf", Sha256Hash: "bbb", Size: 4},
		3: {ID: 3, FileName: "big.bin", Sha256Hash: "ccc", Size: maxSize + 1},
	}
	p := &peer{served: map[int64]string{}}
	p.env = &EnvMock{
		LoggerFunc: func() logger.Logger { return &logger.DummyLogger{} },
		FederationSecretByKIDFunc: func(_ context.Context, k string) (db.FederationSecret, bool, error) {
			return db.FederationSecret{ID: 9, Kid: k, SecretCrypt: []byte(secretKey)}, k == kid, nil
		},
		DecryptDataFunc: func(b []byte) ([]byte, error) { return b, nil },
		ListFederationSecretSubgraphsByKIDFunc: func(context.Context, string) ([]string, error) {
			return []string{"shared"}, nil
		},
		LatestNoteViewsFunc:       func() *model.NoteViews { return nvs },
		FederatedAssetMaxSizeFunc: func() int64 { return maxSize },
		NoteAssetByIDFunc: func(_ context.Context, id int64) (db.NoteAsset, error) {
			row, ok := rows[id]
			if !ok {
				return db.NoteAsset{}, sql.ErrNoRows
			}
			return row, nil
		},
		StreamAssetObjectFunc: func(_ context.Context, asset db.NoteAsset, _, _ int64) (io.ReadCloser, error) {
			p.served[asset.ID] = asset.FileName
			return io.NopCloser(strings.NewReader(sharedZip)), nil
		},
	}

	ln := fasthttputil.NewInmemoryListener()
	endpoint := &federationasset.Endpoint{}
	server := &fasthttp.Server{Handler: func(ctx *fasthttp.RequestCtx) {
		require.Equal(t, endpoint.Path(), string(ctx.Path()))
		req := appreq.Acquire()
		defer appreq.Release(req)
		req.Env = p.env
		req.Req = ctx
		_, err := endpoint.Handle(req)
		if err != nil {
			ctx.SetStatusCode(http.StatusInternalServerError)
		}
	}}
	go func() {
		_ = server.Serve(ln)
	}()
	t.Cleanup(func() {
		_ = ln.Close()
	})

	p.client = &fasthttp.Client{
		Dial:               func(string) (net.Conn, error) { return ln.Dial() },
		StreamResponseBody: true,
		// Small enough that any real attachment streams instead of buffering.
		MaxResponseBodySize: 16,
	}
	return p
}

func (p *peer) assetClient(secret string, maxSize int64) *federation.AssetClient {
	pairing := model.FederationPeer{KBURL: peerURL, KID: kid, Secret: []byte(secret), Issuer: "http://consumer.test"}
	limits := federation.AssetLimits{MaxSize: maxSize, Timeout: 5 * time.Second}
	return federation.NewAssetClient(pairing, p.client, limits)
}

func requireStatus(t *testing.T, err error, status int) {
	t.Helper()

	var httpErr *model.FederationHTTPError
	require.ErrorAs(t, err, &httpErr)
	require.Equal(t, status, httpErr.Status)
}

func TestGrantedSubgraphStreamsTheAsset(t *testing.T) {
	p := newPeer(t)
	client := p.assetClient(secretKey, 1<<20)
	params := model.FederatedAssetParams{NotePath: "skills/reel.md", Name: "files/attach.zip"}

	info, err := client.DescribeAsset(context.Background(), params)
	require.NoError(t, err)
	require.Equal(t, model.NoteAssetInfo{Name: "files/attach.zip", Size: int64(len(sharedZip)), Sha256: "aaa"}, info)
	require.Empty(t, p.served, "describing an asset must not open its bytes")

	asset, err := client.FetchAsset(context.Background(), params)
	require.NoError(t, err)
	defer asset.Body.Close()

	got, err := io.ReadAll(asset.Body)
	require.NoError(t, err)
	require.Equal(t, sharedZip, string(got))
	require.Equal(t, int64(len(sharedZip)), asset.Size)
}

func TestRefusals(t *testing.T) {
	cases := []struct {
		name   string
		params model.FederatedAssetParams
		status int
	}{
		{"a note in a subgraph the pairing was not granted", model.FederatedAssetParams{NotePath: "private/plan.md", Name: "plan.pdf"}, http.StatusNotFound},
		{"a name the note does not have", model.FederatedAssetParams{NotePath: "skills/reel.md", Name: "attach.zip"}, http.StatusNotFound},
		{"another note's asset asked through a granted note", model.FederatedAssetParams{NotePath: "skills/reel.md", Name: "plan.pdf"}, http.StatusNotFound},
		{"an unknown note", model.FederatedAssetParams{NotePath: "nope.md", Name: "files/attach.zip"}, http.StatusNotFound},
		{"an asset over the peer's limit", model.FederatedAssetParams{NotePath: "skills/reel.md", Name: "big.bin"}, http.StatusRequestEntityTooLarge},
		{"no name", model.FederatedAssetParams{NotePath: "skills/reel.md"}, http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := newPeer(t)
			client := p.assetClient(secretKey, 1<<30)

			_, err := client.DescribeAsset(context.Background(), tc.params)
			requireStatus(t, err, tc.status)

			_, err = client.FetchAsset(context.Background(), tc.params)
			requireStatus(t, err, tc.status)

			require.Empty(t, p.served)
		})
	}
}

func TestTheConsumerLimitHoldsWhateverThePeerAllows(t *testing.T) {
	p := newPeer(t)
	client := p.assetClient(secretKey, 8)
	params := model.FederatedAssetParams{NotePath: "skills/reel.md", Name: "files/attach.zip"}

	_, err := client.DescribeAsset(context.Background(), params)
	require.ErrorIs(t, err, model.ErrFederatedAssetTooLarge)

	_, err = client.FetchAsset(context.Background(), params)
	require.ErrorIs(t, err, model.ErrFederatedAssetTooLarge)
}

// The consumer rotated to a key the peer has not confirmed yet: the first
// request is refused and the retry with the previous key gets the file.
func TestARotationInFlightFallsBackToThePreviousKey(t *testing.T) {
	p := newPeer(t)
	pairing := model.FederationPeer{
		KBURL:      peerURL,
		KID:        kid,
		Secret:     []byte("ffffffffffffffffffffffffffffffff"),
		PrevSecret: []byte(secretKey),
		Issuer:     "http://consumer.test",
	}
	client := federation.NewAssetClient(pairing, p.client, federation.AssetLimits{MaxSize: maxSize, Timeout: 5 * time.Second})

	asset, err := client.FetchAsset(context.Background(), model.FederatedAssetParams{NotePath: "skills/reel.md", Name: "files/attach.zip"})
	require.NoError(t, err)
	defer asset.Body.Close()

	got, err := io.ReadAll(asset.Body)
	require.NoError(t, err)
	require.Equal(t, sharedZip, string(got))
}

func TestAnUnknownKeyIsRefused(t *testing.T) {
	p := newPeer(t)
	client := p.assetClient("ffffffffffffffffffffffffffffffff", 1<<20)

	_, err := client.FetchAsset(context.Background(), model.FederatedAssetParams{NotePath: "skills/reel.md", Name: "files/attach.zip"})
	requireStatus(t, err, http.StatusUnauthorized)
	require.Empty(t, p.served)
}

// A token whose signature covers no body is what the pairing-description GET
// carries. Accepting it here would let anyone who saw one name any attachment
// the pairing can read while it lives.
func TestATokenThatDoesNotBindTheBodyIsRefused(t *testing.T) {
	p := newPeer(t)

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	body, err := json.Marshal(model.FederatedAssetParams{NotePath: "skills/reel.md", Name: "files/attach.zip"})
	require.NoError(t, err)

	req.SetRequestURI(peerURL + model.FederatedAssetPath)
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.Set("Authorization", "Bearer "+unboundToken(t))
	req.SetBody(body)

	require.NoError(t, p.client.Do(req, resp))
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode())
	require.Empty(t, p.served)
}

func unboundToken(t *testing.T) string {
	t.Helper()

	now := time.Now().Unix()
	header, err := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT", "kid": kid})
	require.NoError(t, err)
	claims, err := json.Marshal(struct {
		Iss string `json:"iss"`
		Rid string `json:"rid"`
		Iat int64  `json:"iat"`
		Exp int64  `json:"exp"`
	}{Iss: "http://consumer.test", Rid: "1", Iat: now, Exp: now + 30})
	require.NoError(t, err)

	unsigned := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(claims)
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(unsigned))
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func TestNoNotesLoadedYetIsNotFound(t *testing.T) {
	p := newPeer(t)
	p.env.LatestNoteViewsFunc = func() *model.NoteViews { return nil }
	params := model.FederatedAssetParams{NotePath: "skills/reel.md", Name: "files/attach.zip"}

	_, err := federationasset.Resolve(context.Background(), p.env, mcp.FederationCaller{}, params)
	require.ErrorIs(t, err, federationasset.ErrAssetNotFound)
}
