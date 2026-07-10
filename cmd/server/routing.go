package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
	"trip2g/internal/appreq"
	"trip2g/internal/case/signinbytgauthtoken"
	"trip2g/internal/model"
	"trip2g/internal/readreplica"
	"trip2g/internal/rssfeed"

	"github.com/valyala/fasthttp"
)

func (a *app) GetPublicURLForRequest(ctx context.Context) string {
	// If PublicURL is configured, use it
	if publicURL := a.config.PublicURL; publicURL != "" {
		return publicURL
	}

	// Otherwise, extract URL from the current request
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		// Fallback to empty string if no request context
		return ""
	}

	if req.Req == nil {
		return ""
	}

	// Get scheme (http or https)
	scheme := "http"
	if req.Req.IsTLS() {
		scheme = "https"
	}

	// Get host from request
	host := string(req.Req.Host())

	return fmt.Sprintf("%s://%s", scheme, host)
}

func (a *app) TrustedDomains() []string {
	domains := []string{}

	// Always add the public URL domain
	if publicURL := a.config.PublicURL; publicURL != "" {
		if u, err := url.Parse(publicURL); err == nil && u.Host != "" {
			domains = append(domains, u.Host)
		}
	}

	// In dev mode, also add localhost:8081
	if a.config.DevMode {
		domains = append(domains, "localhost:8081")
	}

	return domains
}

func (a *app) RefreshNotFoundTracker(ctx context.Context) error {
	if a.notFoundTracker == nil {
		return nil
	}
	return a.notFoundTracker.Refresh(ctx)
}

func (a *app) TrackNotFound(path string, ip string) {
	if a.config.DevMode {
		a.log.Warn("page not found", "path", path)
	}

	if a.notFoundTracker == nil {
		return
	}

	if !a.SiteConfig(context.Background()).EnableNotFoundTracking {
		return
	}

	err := a.notFoundTracker.Track(path, ip)
	if err != nil {
		a.log.Error("failed to track not found", "path", path, "error", err)
	}
}

func (a *app) RequestIP(ctx context.Context) string {
	req, err := appreq.FromCtx(ctx)
	if err != nil {
		return ""
	}
	// Check X-Real-IP / X-Forwarded-For first (behind proxy)
	if ip := string(req.Req.Request.Header.Peek("X-Real-IP")); ip != "" {
		return ip
	}
	if ip := string(req.Req.Request.Header.Peek("X-Forwarded-For")); ip != "" {
		return ip
	}
	ip := req.Req.RemoteIP()
	if ip == nil {
		return ""
	}
	return ip.String()
}

func (a *app) handleCors(ctx *fasthttp.RequestCtx) bool {
	origin := string(ctx.Request.Header.Peek("Origin"))
	if origin == "http://localhost:9081" || origin == "app://obsidian.md" {
		ctx.Response.Header.Set("Access-Control-Allow-Origin", origin)
		ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
		ctx.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type, Cookie, X-API-Key, X-Plugin-Version")
		ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
	}

	if ctx.IsOptions() {
		ctx.SetStatusCode(fasthttp.StatusNoContent)
		return true
	}

	return false
}

func (a *app) handleRSSFeed(req *appreq.Request) bool {
	if !strings.HasSuffix(req.Path, ".rss.xml") {
		return false
	}

	cfg := a.SiteConfig(context.Background())
	if !cfg.EnableRSS {
		return false
	}

	// Strip .rss.xml suffix to get the note path.
	notePath := strings.TrimSuffix(req.Path, ".rss.xml")
	if notePath == "" {
		notePath = "/"
	}

	notes := a.LiveNoteViews()
	note := notes.GetByPath(notePath)
	// RSS is an unauthenticated endpoint and must honor every static anonymous
	// access gate, including sign-in-only subgraphs and internal/template notes.
	if !rssfeed.IsPubliclyReadable(note) {
		return false
	}

	xmlBytes, err := rssfeed.Generate(note, a.PublicURL(), notes)
	if err != nil {
		a.log.Error("failed to generate RSS feed", "error", err, "path", req.Path)
		return false
	}

	req.Req.SetContentType("application/rss+xml; charset=utf-8")
	req.Req.SetStatusCode(http.StatusOK)
	req.Req.SetBody(xmlBytes)

	return true
}

func (a *app) handleSitemap(req *appreq.Request) bool {
	if req.Path != "/sitemap.xml" {
		return false
	}

	// Check if this is a custom domain request.
	host := model.NormalizeDomain(string(req.Req.Host()))
	mainHost := model.NormalizeDomain(model.ExtractHost(a.PublicURL()))
	if host != mainHost && host != "" {
		nvs := a.LiveNoteViews()
		if nvs != nil {
			if domainSitemap, ok := nvs.DomainSitemaps[host]; ok && len(domainSitemap) > 0 {
				req.Req.SetContentType("application/xml; charset=utf-8")
				req.Req.SetStatusCode(http.StatusOK)
				req.Req.Response.SetBody(domainSitemap)
				return true
			}
		}
	}

	nvs := a.LiveNoteViews()
	if nvs == nil || len(nvs.Sitemap) == 0 {
		return false
	}

	req.Req.SetContentType("application/xml; charset=utf-8")
	req.Req.SetStatusCode(http.StatusOK)
	req.Req.SetBody(nvs.Sitemap)

	return true
}

// Middleware should return true if the request is fully handled.
type Middleware func(req *appreq.Request) bool

func (a *app) prepareMiddlewares() []Middleware {
	fsHandler := a.assetsFS.NewRequestHandler()

	// Local-storage asset handler: serves GET /_assets/<NoteAssetPath> from disk.
	// nil unless the local storage backend is active.
	var localAssetsHandler fasthttp.RequestHandler
	if a.localAssetsFS != nil {
		localAssetsHandler = a.localAssetsFS.NewRequestHandler()
	}

	return []Middleware{
		// Read-only replica: forward every mutating request to the leader before
		// any local handler runs (so no handler side effects are replayed). Safe
		// methods (GET/HEAD/OPTIONS) fall through and are served locally.
		func(req *appreq.Request) bool {
			if a.replicaForwarder == nil {
				return false
			}
			if !readreplica.IsWrite(string(req.Req.Method())) {
				return false
			}
			a.replicaForwarder.Forward(req.Req)
			return true
		},
		a.handleSitemap,
		a.handleRSSFeed,
		func(req *appreq.Request) bool {
			return a.handleCors(req.Req)
		},
		func(req *appreq.Request) bool {
			return a.handleDebugAPI(req.Req)
		},
		func(req *appreq.Request) bool {
			return a.gitAPI != nil && a.gitAPI.HandleRequest(req.Req)
		},
		func(req *appreq.Request) bool {
			if strings.HasPrefix(req.Path, "/assets/") {
				fsHandler(req.Req)
				return true
			}

			return false
		},
		func(req *appreq.Request) bool {
			if localAssetsHandler != nil && strings.HasPrefix(req.Path, "/_assets/") {
				localAssetsHandler(req.Req)
				return true
			}

			return false
		},
		func(req *appreq.Request) bool {
			return a.handlePurchaseTokens(req.Req)
		},
		func(req *appreq.Request) bool {
			return signinbytgauthtoken.Process(req.Req, a)
		},
		func(req *appreq.Request) bool {
			secret := string(req.Req.Request.Header.Peek("X-Telegram-Bot-Api-Secret-Token"))
			handled, status := a.TgBots.ProcessWebhookRequest(req.Path, secret, func() []byte { return req.Req.PostBody() })
			if handled {
				req.Req.SetStatusCode(status)
			}
			return handled
		},
	}
}

// handleReplicaIntake authenticates a forwarded write from a read replica and
// runs it through the full app handler. Reject (401) on a missing/invalid
// X-Replica-Auth; 503 until the app handler is built (leader still warming up).
func (a *app) handleReplicaIntake(ctx *fasthttp.RequestCtx) {
	auth := string(ctx.Request.Header.Peek(readreplica.AuthHeader))
	if err := readreplica.VerifyAuth(a.config.UserToken.Secret, auth, time.Now()); err != nil {
		a.log.Warn("replica intake: rejected request", "error", err, "remote", ctx.RemoteIP().String(), "path", string(ctx.Path()))
		ctx.SetStatusCode(http.StatusUnauthorized)
		ctx.SetBodyString("401 replica intake: invalid X-Replica-Auth")
		return
	}

	h := a.appHandler.Load()
	if h == nil {
		ctx.SetStatusCode(http.StatusServiceUnavailable)
		ctx.SetBodyString("503 replica intake: leader not ready")
		return
	}

	(*h)(ctx)
}
