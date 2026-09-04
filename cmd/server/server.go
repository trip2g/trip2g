package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"net/url"
	"strings"
	"time"
	"trip2g/internal/acmecache"
	"trip2g/internal/appreq"
	"trip2g/internal/appresp"
	"trip2g/internal/case/signinbyhat"
	"trip2g/internal/metrics"
	"trip2g/internal/noteloader"
	"trip2g/internal/router"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

// serveHTTP serves the public listener, preferring a systemd socket-activated
// listener (so the socket survives restarts and queues connections during
// warmup) and falling back to a fresh bind on --listen-addr.
func (a *app) serveHTTP(s *fasthttp.Server) error {
	if ln := a.systemdListener(); ln != nil {
		return s.Serve(ln)
	}

	return s.ListenAndServe(a.config.ListenAddr)
}

func (a *app) startServer() { //nolint:gocognit // server startup wiring
	makeGraphQLHandler := a.prepareGraphQLHandler()
	handleGraphQL := makeGraphQLHandler("/_system/graphql")
	handleGraphQLCompatRaw := makeGraphQLHandler("/graphql")

	// The deprecated /graphql alias keeps working until a live note at /graphql
	// replaces it — then the note takes over the path (served by the router's
	// note renderer) and GraphQL stays at its canonical /_system/graphql.
	handleGraphQLCompat := func(ctx *fasthttp.RequestCtx, path string) bool {
		if !strings.HasPrefix(path, "/graphql") {
			return false
		}

		if nvs := a.LiveNoteViews(); nvs != nil && nvs.GetByPath("/graphql") != nil {
			return false
		}

		return handleGraphQLCompatRaw(ctx, path)
	}

	rtr := router.New(a)

	middlewares := a.prepareMiddlewares()

	handler := func(ctx *fasthttp.RequestCtx) {
		path := string(ctx.Path())

		req := appreq.Acquire()
		req.Env = a
		req.Req = ctx
		req.Path = path
		req.TokenManager = a.tokenManager
		req.PersonalTokenResolver = a.personalTokenResolver
		req.Client = strings.TrimSpace(string(ctx.Request.Header.Peek("X-trip2g-client")))
		req.StoreInContext() // appreq.FromCtx(ctx)
		defer appreq.Release(req)

		for _, mw := range middlewares {
			if mw(req) {
				return
			}
		}

		// Hot auth token consumer: a buyer redirected back after payment with
		// ?hat=<token> gets auto-logged-in. Resolve sets the trip2g_token cookie
		// on this response (via SetupUserToken -> TokenManager.Store), then we
		// strip ?hat and redirect to the cleaned URL so the token doesn't linger
		// in the address bar. A bad/expired token is logged and falls through so
		// it never breaks the return page.
		if hatAuthToken := string(ctx.QueryArgs().Peek("hat")); hatAuthToken != "" {
			if _, hatErr := signinbyhat.Resolve(ctx, a, hatAuthToken); hatErr != nil {
				a.log.Warn("failed to resolve hot auth token", "error", hatErr)
			} else if parsedURL, parseErr := url.Parse(string(ctx.Request.Header.RequestURI())); parseErr != nil {
				a.log.Warn("failed to parse URL after hat sign-in", "error", parseErr)
			} else {
				query := parsedURL.Query()
				query.Del("hat")
				parsedURL.RawQuery = query.Encode()

				appresp.Redirect(ctx, parsedURL.String(), http.StatusFound)
				return
			}
		}

		if handleGraphQL(ctx, path) {
			return
		}

		if handleGraphQLCompat(ctx, path) {
			return
		}

		newPath := a.redirectManager.Match(path)
		if newPath != nil {
			ctx.SetStatusCode(http.StatusFound)
			ctx.Response.Header.Set("Location", *newPath)
			return
		}

		handled, handleErr := rtr.Handle(req)
		if handleErr != nil {
			a.log.Error("failed to handle request", "error", handleErr)
			ctx.SetStatusCode(http.StatusServiceUnavailable)
			ctx.SetBodyString("500 Internal Server Error")
			return
		}

		// TODO: remove this code because rendernotepage handles 404
		if handled {
			a.log.Debug("router handled request", "path", path)
			return
		}

		if handleGraphQLCompat(ctx, path) {
			return
		}

		ctx.SetStatusCode(http.StatusNotFound)
		ctx.SetBodyString("404 Not Found")
	}

	handlerTimeout := 60 * time.Second
	if a.config.DevMode {
		handlerTimeout = 10 * time.Minute
	}

	timeoutHandler := fasthttp.TimeoutHandler(handler, handlerTimeout, "timeout")
	compressedHandler := fasthttp.CompressHandler(timeoutHandler)

	s := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			// SSE connections are long-lived, skip timeout handler for them.
			if strings.Contains(string(ctx.Request.Header.Peek("Accept")), "text/event-stream") {
				handler(ctx)
				return
			}
			compressedHandler(ctx)
		},
		ErrorHandler: func(ctx *fasthttp.RequestCtx, err error) {
			if errors.Is(err, fasthttp.ErrBodyTooLarge) {
				ctx.SetStatusCode(http.StatusRequestEntityTooLarge)
				ctx.SetContentType("application/json")
				ctx.SetBodyString(`{"errors":[{"message":"request body size exceeds the limit"}]}`)
				return
			}
			// Default behavior for other errors.
			ctx.Error("Error when parsing request", fasthttp.StatusBadRequest)
		},
		MaxRequestBodySize: a.config.MaxRequestBodySize * 1024 * 1024,
		ReadTimeout:        handlerTimeout,
		WriteTimeout:       handlerTimeout,
		IdleTimeout:        handlerTimeout,
	}

	// Publish the full request handler so the internal server's leader-side
	// replica intake (handleReplicaIntake) can run forwarded writes through the
	// real pipeline on --internal-listen-addr. Until this point it returns 503.
	appHandler := s.Handler
	a.appHandler.Store(&appHandler)

	runServer := func() {
		if len(a.config.AcmeDomains) == 0 {
			a.log.Info("http server listening", "addr", a.config.ListenAddr)
			if err := a.serveHTTP(s); err != nil {
				panic(err)
			}

			return
		}

		a.startACMEServer(s)
	}

	if a.config.DevMode {
		runServer()
	} else {
		go runServer()
		a.waitForShutdown(s)
	}
}

func (a *app) waitForShutdown(s *fasthttp.Server) {
	<-a.sigChan

	a.stopped.Store(true)
	a.shutdown() // notify all shutdown listeners

	a.log.Info("shutting down in", "grace_period", a.config.ShutdownGracePeriod)

	time.Sleep(a.config.ShutdownGracePeriod)

	// Perform shutdown backup if enabled AND this is not a zero-downtime handoff.
	// In a rolling deploy a peer is taking over: the departing instance's backup
	// is redundant (cron backups continue and the new writer is live) and the dump
	// would race the new writer / delay the drain. Gate with
	// --simple-backup-on-shutdown=false in the rolling path.
	if a.simpleBackup != nil && a.config.SimpleBackup.BackupOnShutdown {
		a.log.Info("performing shutdown backup...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := a.simpleBackup.PerformBackup(ctx); err != nil {
			a.log.Error("shutdown backup failed", "error", err)
		} else {
			a.log.Info("shutdown backup completed")
		}
	}

	// Stop writer subsystems BEFORE draining HTTP and BEFORE releasing the
	// writer slot. a.stopped is already true (set at the top of shutdown), so
	// /readyz is already 503 and traffic is being routed away. Stopping writers
	// here lets the NEXT instance acquire the writer slot for handoff.
	a.stopWriters()

	a.log.Info("shutting down server", "timeout", a.config.ShutdownTimeout)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), a.config.ShutdownTimeout)
	defer shutdownCancel()

	err := s.ShutdownWithContext(shutdownCtx)
	if err != nil {
		a.log.Error("failed to shutdown server gracefully", "error", err)
	}

	if a.internalServer != nil {
		if internalErr := a.internalServer.Shutdown(); internalErr != nil {
			a.log.Error("failed to shutdown internal server", "error", internalErr)
		}
	}

	// Release the search indexes last. With an on-disk index this unlocks the
	// directory for the next instance and leaves it in a state that reopens in
	// milliseconds; with the default in-memory index it is a no-op.
	for _, loader := range []*noteloader.Loader{a.liveNoteLoader, a.latestNoteLoader} {
		if loader == nil {
			continue
		}
		if closeErr := loader.Close(); closeErr != nil {
			a.log.Error("failed to close search index", "error", closeErr)
		}
	}

	a.log.Info("server stopped")
}

// stopWriters stops every writer subsystem started in Block B: the background
// job queues, the cron scheduler, and the patreon/boosty refresh loops. It then
// releases the writer slot. With the current probe approach there is no held
// lock to release explicitly — stopping the writers is what frees the SQLite
// write lock for the next instance. Each field is nil-checked because SIGTERM
// can arrive before Block B finished (e.g. during read-only warmup).
func (a *app) stopWriters() {
	a.log.Info("stopping writer subsystems")

	// Stop and wait for the queue runners so in-flight writing jobs finish.
	for _, q := range a.appQueues {
		q.stopAndWait()
	}

	if a.CronJobs != nil {
		a.CronJobs.StopCronJobs()
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if a.PatreonJobs != nil {
		a.PatreonJobs.Stop(stopCtx)
	}

	if a.BoostyJobs != nil {
		a.BoostyJobs.Stop(stopCtx)
	}

	// Writer slot release is a no-op for the probe approach: the SQLite write
	// lock is freed simply by the writers above having stopped. Phase 2/Consul
	// will add an explicit release here.
}

// isReady reports whether the instance can fully serve, including writes. It is
// false while warming up (writer slot not yet acquired / writer subsystems not
// started) and while shutting down. Backs the /readyz readiness endpoint.
func (a *app) isReady() bool {
	return !a.stopped.Load() && a.ready.Load()
}

func (a *app) startInternalServer() {
	// Metrics, pprof and the embedding debug endpoint keep their net/http
	// handlers; they're mounted on a mux and adapted to fasthttp once below. The
	// internal server itself is fasthttp so the leader-side replica intake can
	// run forwarded writes through the real (fasthttp) app handler on the same
	// port — no separate listener, no net/http→fasthttp request bridge.
	debugMux := http.NewServeMux()

	// Register DB observability collectors (pool stats + file metrics).
	metrics.RegisterDBCollectors(a.conn, a.writeConn, a.queueConn, a.config.DatabaseFile)

	// Prometheus metrics endpoint
	debugMux.Handle("/metrics", metrics.Setup())

	// Register pprof endpoints
	debugMux.HandleFunc("/debug/pprof/", pprof.Index)
	debugMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	debugMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	debugMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	debugMux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	debugMux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	debugMux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	debugMux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	debugMux.Handle("/debug/pprof/block", pprof.Handler("block"))
	debugMux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
	debugMux.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))

	// Embedding debug endpoint — step-by-step pipeline: split → embed → JSON result.
	if a.debugEmbeddingEnabled() {
		debugMux.HandleFunc("/debug/embedding", a.handleDebugEmbedding)
	}

	// Write-holder diagnostic: names who occupies the single write connection.
	// Dev-only — the tracker captures stacks per acquire, too costly for prod.
	if a.config.DevMode {
		debugMux.HandleFunc("/debug/write-holder", a.handleDebugWriteHolder)
	}

	debugHandler := fasthttpadaptor.NewFastHTTPHandler(debugMux)

	// Start metrics updater
	metricsUpdater := metrics.NewUpdater(a, a.config.Metrics.UpdateInterval, prometheus.DefaultRegisterer)
	go func() {
		err := metricsUpdater.Run(a.ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			a.log.Error("metrics updater stopped", "error", err)
		}
	}()

	handler := func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/healthz":
			if a.stopped.Load() {
				ctx.SetStatusCode(http.StatusServiceUnavailable)
				ctx.SetBodyString("shutting down")
				return
			}
			ctx.SetStatusCode(http.StatusOK)
			ctx.SetBodyString("ok")
			return
		// /livez is liveness (k8s convention): 200 whenever the process can
		// answer, regardless of warmup or shutdown state. A warming or draining
		// instance is still ALIVE and must NOT be restarted.
		case "/livez":
			ctx.SetStatusCode(http.StatusOK)
			ctx.SetBodyString("alive")
			return
		// /readyz is readiness: 200 only when the instance can fully serve. 503
		// while warming up or shutting down so routing/deploy gating is correct.
		case "/readyz":
			if !a.isReady() {
				ctx.SetStatusCode(http.StatusServiceUnavailable)
				ctx.SetBodyString("not ready")
				return
			}
			ctx.SetStatusCode(http.StatusOK)
			ctx.SetBodyString("ready")
			return
		}

		path := string(ctx.Path())
		if path == "/metrics" || strings.HasPrefix(path, "/debug/") {
			debugHandler(ctx)
			return
		}

		// Leader-side replica intake: any other path is a forwarded write from a
		// replica. Authenticate it (HMAC over --jwt-secret) and run it through the
		// real app pipeline. Replicas never accept intake (they forward, not host).
		if !a.config.IsReadReplica() {
			a.handleReplicaIntake(ctx)
			return
		}

		ctx.SetStatusCode(http.StatusNotFound)
	}

	a.internalServer = &fasthttp.Server{
		Handler: handler,
		Name:    "trip2g-internal",
	}

	a.log.Info("starting internal server", "addr", a.config.InternalListenAddr)
	if err := a.internalServer.ListenAndServe(a.config.InternalListenAddr); err != nil {
		panic(err)
	}
}

func (a *app) startACMEServer(s *fasthttp.Server) {
	allowedDomains := make(map[string]struct{}, len(a.config.AcmeDomains))

	for _, domain := range a.config.AcmeDomains {
		a.log.Info("adding domain to ACME", "domain", domain)
		allowedDomains[domain] = struct{}{}
	}

	certManager := autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  acmecache.New(a),
		HostPolicy: func(ctx context.Context, host string) error {
			_, ok := allowedDomains[host]
			if ok {
				return nil
			}

			return fmt.Errorf("unauthorized domain: %s", host)
		},
	}

	// Start HTTP server on port 80 for ACME challenges and HTTPS redirect
	httpServer := &http.Server{
		Addr:         ":80",
		Handler:      certManager.HTTPHandler(nil),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	go func() {
		<-a.shutdownCtx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.config.ShutdownTimeout)
		defer cancel()

		err := httpServer.Shutdown(shutdownCtx)
		if err != nil {
			a.log.Error("failed to shutdown HTTP redirect server", "error", err)
		}
	}()

	go func() {
		a.log.Info("starting HTTP redirect server on port 80")
		err := httpServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			a.log.Error("HTTP redirect server failed", "error", err)
		}
	}()

	tlsConfig := &tls.Config{
		GetCertificate: certManager.GetCertificate,
		NextProtos:     []string{"http/1.1", acme.ALPNProto},
		MinVersion:     tls.VersionTLS12,
	}

	ln, err := net.Listen("tcp4", ":443") // #nosec G102
	if err != nil {
		panic(err)
	}

	lnTLS := tls.NewListener(ln, tlsConfig)

	a.log.Info("starting HTTPS server on port 443")
	err = fasthttp.Serve(lnTLS, s.Handler)
	if err != nil {
		panic(err)
	}
}
