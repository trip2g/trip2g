package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"trip2g/internal/case/mcp"
	"trip2g/internal/case/system/federatedassetrelay"
	"trip2g/internal/db"
	"trip2g/internal/federatedassetlink"
	"trip2g/internal/federation"
	"trip2g/internal/metrics"
	"trip2g/internal/model"
	"trip2g/internal/ssrfsafe"

	"github.com/valyala/fasthttp"
)

func (a *app) FederationSecretByKBURL(ctx context.Context, kbURL string) (db.FederationSecret, bool, error) {
	secret, err := a.Queries.FederationSecretByKBURL(ctx, &kbURL)
	if errors.Is(err, sql.ErrNoRows) {
		return db.FederationSecret{}, false, nil
	}
	if err != nil {
		return db.FederationSecret{}, false, err
	}
	return secret, true, nil
}

func (a *app) FederationSecretByKID(ctx context.Context, kid string) (db.FederationSecret, bool, error) {
	secret, err := a.Queries.FederationSecretByKID(ctx, kid)
	if errors.Is(err, sql.ErrNoRows) {
		return db.FederationSecret{}, false, nil
	}
	if err != nil {
		return db.FederationSecret{}, false, err
	}
	return secret, true, nil
}

func (a *app) HasFederationSecretForKBURL(ctx context.Context, kbURL string) (bool, error) {
	rows, err := a.Queries.ListFederationSecrets(ctx)
	if err != nil {
		return false, err
	}
	for _, row := range rows {
		if row.KbUrl != nil && *row.KbUrl == kbURL {
			return true, nil
		}
	}
	return false, nil
}

func (a *app) FederationClient(reqCtx context.Context, kbID string) (model.Federation, error) {
	peer, err := a.federationPeer(reqCtx, kbID)
	if err != nil {
		return nil, err
	}
	return federation.NewClient(peer, a.fedHTTPClient, a.config.DevMode, a.config.MCPFederationTimeout), nil
}

func (a *app) FederationAssetFetcher(reqCtx context.Context, kbID string) (model.FederationAssetFetcher, error) {
	peer, err := a.federationPeer(reqCtx, kbID)
	if err != nil {
		return nil, err
	}
	limits := federation.AssetLimits{
		MaxSize: a.FederatedAssetMaxSize(),
		Timeout: a.config.MCPFederationAssetTimeout,
	}
	return federation.NewAssetClient(peer, a.fedAssetHTTPClient, limits), nil
}

func (a *app) FederatedAssetMaxSize() int64 {
	return int64(a.config.MCPFederationAssetMaxSize.Bytes())
}

func (a *app) FederatedAssetLinkSecret() string {
	return a.config.UserToken.Secret
}

// FederatedAssetDownload issues the link federated_asset hands its caller.
func (a *app) FederatedAssetDownload(kbID string, params model.FederatedAssetParams) (mcp.FederatedAssetDownload, error) {
	expiresAt := time.Now().Add(federatedAssetLinkTTL)
	link := federatedassetlink.Link{KBID: kbID, NotePath: params.NotePath, Name: params.Name}

	token, err := federatedassetlink.Sign(link, a.FederatedAssetLinkSecret(), expiresAt)
	if err != nil {
		return mcp.FederatedAssetDownload{}, err
	}

	query := url.Values{federatedassetrelay.LinkParam: {token}}
	download := mcp.FederatedAssetDownload{
		URL:       strings.TrimRight(a.PublicURL(), "/") + federatedassetrelay.Path + "?" + query.Encode(),
		ExpiresAt: expiresAt,
	}
	return download, nil
}

// federatedAssetLinkTTL is long enough for an agent to run curl after the tool
// call, and short enough that a link left in a transcript is soon worthless.
const federatedAssetLinkTTL = 5 * time.Minute

// federationPeer is the outbound side of the pairing kbID names: the KB-note's
// URL and the keys this instance signs with.
func (a *app) federationPeer(reqCtx context.Context, kbID string) (model.FederationPeer, error) {
	nvs := a.LatestNoteViews()
	if nvs == nil {
		return model.FederationPeer{}, fmt.Errorf("federation kb %q not found", kbID)
	}

	depth := mcp.FederationDepthFromContext(reqCtx)

	for _, kb := range nvs.MCPFederationNotes {
		if kb == nil || kb.ID != kbID {
			continue
		}

		if kb.MaxDepth > 0 && depth >= kb.MaxDepth {
			return model.FederationPeer{}, fmt.Errorf("federation kb %q max depth %d exceeded", kbID, kb.MaxDepth)
		}

		peer := model.FederationPeer{
			KBID:   kb.ID,
			KBURL:  kb.URL,
			Issuer: a.PublicURL(),
			Depth:  depth,
		}

		dbCtx := a.ctx
		if dbCtx == nil {
			dbCtx = context.Background()
		}
		secretRow, ok, err := a.FederationSecretByKBURL(dbCtx, kb.URL)
		if err != nil {
			return model.FederationPeer{}, fmt.Errorf("get federation secret by kb url: %w", err)
		}
		if ok {
			keyErr := a.loadPeerKeys(&peer, secretRow)
			if keyErr != nil {
				return model.FederationPeer{}, keyErr
			}
		} else {
			configured, confErr := a.HasFederationSecretForKBURL(dbCtx, kb.URL)
			if confErr != nil {
				return model.FederationPeer{}, fmt.Errorf("check federation secret history by kb url: %w", confErr)
			}
			if configured {
				return model.FederationPeer{}, fmt.Errorf("no active federation secret for kb_id %q; the configured secret may be revoked", kbID)
			}
		}

		return peer, nil
	}

	return model.FederationPeer{}, fmt.Errorf("federation kb %q not found", kbID)
}

func (a *app) FederationMaxDepth() int {
	return a.config.MCPFederationMaxDepth
}

func (a *app) FederatedFanoutConcurrency() int {
	return a.config.FederatedFanoutConcurrency
}

func (a *app) FederatedFanoutLimit() int {
	return a.config.FederatedFanoutLimit
}

func (a *app) FederatedFanoutTimeout() time.Duration {
	return a.config.FederatedFanoutTimeout
}

func (a *app) MCPMetrics() *metrics.MCPMetrics {
	return a.mcpMetrics
}

func (a *app) RotateFederationSecret(ctx context.Context, arg db.RotateFederationSecretParams) (int64, error) {
	return a.WriteQueries.RotateFederationSecret(ctx, arg)
}

func (a *app) ClearFederationSecretPrev(ctx context.Context, arg db.ClearFederationSecretPrevParams) error {
	return a.WriteQueries.ClearFederationSecretPrev(ctx, arg)
}

// FederationPeerClient builds a client for a peer described by the caller rather
// than by a KB-note. Rotation needs it: the pairing it talks to is named by a
// secret row, and at install time there is no note and no row yet.
func (a *app) FederationPeerClient(peer model.FederationPeer) model.Federation {
	return federation.NewClient(peer, a.fedHTTPClient, a.config.DevMode, a.config.MCPFederationTimeout)
}

// FederationAllowsPlainHTTP reports whether this deployment federates over
// addresses that are not on the public internet. It is the same condition that
// decides whether the dialer refuses private addresses, and deliberately not a
// setting of its own: an internal address rarely has a certificate, and a second
// flag would only ever repeat what this one already says.
func (a *app) FederationAllowsPlainHTTP() bool {
	return a.config.DevMode || a.config.MCPFederationAllowPrivate
}

// loadPeerKeys puts the pairing's keys on an outbound peer: the current one, and
// the one it rotated away from while the other side could still be holding it.
//
// Outside that window the peer has stopped accepting the old key too, so
// carrying it would buy a second refused request per call and nothing else.
func (a *app) loadPeerKeys(peer *model.FederationPeer, row db.FederationSecret) error {
	secret, err := a.DecryptData(row.SecretCrypt)
	if err != nil {
		return err
	}

	peer.KID = row.Kid
	peer.Secret = secret

	if len(row.PrevSecretCrypt) == 0 || row.RotatedAt == nil {
		return nil
	}
	if time.Since(*row.RotatedAt) > model.RotationGrace {
		return nil
	}

	previous, err := a.DecryptData(row.PrevSecretCrypt)
	if err != nil {
		return err
	}
	peer.PrevSecret = previous

	return nil
}

// newFederatedAssetHTTPClient builds the client attachments stream through.
// fasthttp buffers a streamed response whole when it fits MaxResponseBodySize
// and streams it only beyond that, so the cap here is a buffer size, not a
// limit: the size limit is AssetLimits.MaxSize.
func newFederatedAssetHTTPClient(allowPrivate bool) *fasthttp.Client {
	c := &fasthttp.Client{
		StreamResponseBody:  true,
		MaxResponseBodySize: 64 << 10,
	}
	if !allowPrivate {
		c.DialTimeout = ssrfsafe.DialTimeout
	}
	return c
}
