package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/valyala/fasthttp"

	"trip2g/internal/case/backjob/importtelegramchannel"
	"trip2g/internal/db"
	"trip2g/internal/model"
)

type webhookTestCall struct {
	Timestamp int64             `json:"timestamp"`
	Headers   map[string]string `json:"headers"`
	Body      json.RawMessage   `json:"body"`
}

// debugJobRecord is written by the debug_sleep_job when it completes.
type debugJobRecord struct {
	Tag         string    `json:"tag"`
	DurationMs  int       `json:"durationMs"`
	CompletedAt time.Time `json:"completedAt"`
}

type debugSleepJobParams struct {
	Tag        string `json:"tag"`
	DurationMs int    `json:"durationMs"`
}

const debugSleepJobID = "debug_sleep_job"

// initDebugJobs registers the debug_sleep_job handler. Only active in devMode.
func (a *app) initDebugJobs() {
	if !a.config.DevMode {
		return
	}
	a.RegisterJob(model.BackgroundDefaultQueue, debugSleepJobID, func(_ context.Context, m []byte) error {
		var params debugSleepJobParams
		if err := json.Unmarshal(m, &params); err != nil {
			return fmt.Errorf("unmarshal debug sleep params: %w", err)
		}
		if params.DurationMs > 0 {
			time.Sleep(time.Duration(params.DurationMs) * time.Millisecond)
		}
		a.debugJobMu.Lock()
		a.debugJobLog = append(a.debugJobLog, debugJobRecord{
			Tag:         params.Tag,
			DurationMs:  params.DurationMs,
			CompletedAt: time.Now().UTC(),
		})
		a.debugJobMu.Unlock()
		return nil
	})
}

func (a *app) handleDebugAPI(ctx *fasthttp.RequestCtx) bool {
	if !a.config.DevMode {
		return false
	}

	path := string(ctx.Path())

	switch {
	case ctx.IsPost() && strings.HasPrefix(path, "/debug/test_webhook"):
		return a.handleDebugTestWebhook(ctx)

	case ctx.IsGet() && strings.HasPrefix(path, "/debug/test_webhook_calls"):
		return a.handleDebugTestWebhookCalls(ctx)

	case ctx.IsDelete() && strings.HasPrefix(path, "/debug/test_webhook_calls"):
		return a.handleDebugTestWebhookCallsClear(ctx)

	case strings.HasPrefix(path, "/debug/layouts/latest"):
		return a.handleDebugLayoutsLatest(ctx)

	case strings.HasPrefix(path, "/debug/nvs/subgraphs"):
		return a.handleDebugNvsSubgraphs(ctx)

	case strings.HasPrefix(path, "/debug/nvs/latest"):
		return a.handleDebugNvsLatest(ctx)

	case strings.HasPrefix(path, "/debug/form_spec"):
		return a.handleDebugFormSpec(ctx)

	case strings.HasPrefix(path, "/debug/wait_all_jobs"):
		return a.handleDebugWaitAllJobs(ctx)

	case ctx.IsPost() && strings.HasPrefix(path, "/debug/spawn_jobs"):
		return a.handleDebugSpawnJobs(ctx)

	case ctx.IsGet() && strings.HasPrefix(path, "/debug/completed_jobs"):
		return a.handleDebugCompletedJobs(ctx)

	case ctx.IsDelete() && strings.HasPrefix(path, "/debug/completed_jobs"):
		return a.handleDebugCompletedJobsClear(ctx)

	case strings.HasPrefix(path, "/debug/run_cron_job"):
		return a.handleDebugRunCronJob(ctx)

	case strings.HasPrefix(path, "/debug/telegram_import"):
		return a.handleDebugTelegramImport(ctx)
	}

	return false
}

func (a *app) handleDebugLayoutsLatest(ctx *fasthttp.RequestCtx) bool {
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)

	data, err := json.Marshal(a.Layouts()) //nolint:musttag // debug endpoint, func fields skipped by json
	if err != nil {
		a.log.Error("failed to marshal latest note views", "error", err)
		return true
	}

	ctx.SetBody(data)
	return true
}

type debugSubgraphInfo struct {
	RequireSignin bool `json:"require_signin"`
}

func (a *app) handleDebugNvsSubgraphs(ctx *fasthttp.RequestCtx) bool {
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)

	type result struct {
		Latest map[string]debugSubgraphInfo `json:"latest"`
		Live   map[string]debugSubgraphInfo `json:"live"`
	}

	toDebugMap := func(sgs map[string]*model.NoteSubgraph) map[string]debugSubgraphInfo {
		m := make(map[string]debugSubgraphInfo, len(sgs))
		for k, v := range sgs {
			m[k] = debugSubgraphInfo{RequireSignin: v.RequireSignin}
		}
		return m
	}

	res := result{
		Latest: toDebugMap(a.latestNoteLoader.NoteViews().Subgraphs),
		Live:   toDebugMap(a.liveNoteLoader.NoteViews().Subgraphs),
	}

	data, err := json.Marshal(res)
	if err != nil {
		a.log.Error("failed to marshal subgraphs", "error", err)
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return true
	}
	ctx.SetBody(data)
	return true
}

func (a *app) handleDebugNvsLatest(ctx *fasthttp.RequestCtx) bool {
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)

	type noteDebugInfo struct {
		Path      string `json:"path"`
		VersionID int64  `json:"version_id"`
		PathID    int64  `json:"path_id"`
		Title     string `json:"title"`
		HasForm   bool   `json:"has_form"`
	}

	nvs := a.LatestNoteViews()
	all := nvs.VisibleList()
	notes := make([]noteDebugInfo, 0, len(all))
	for _, nv := range all {
		_, hasForm := nv.RawMeta["form"]
		_, hasForms := nv.RawMeta["forms"]
		_, hasFormRef := nv.RawMeta["form_ref"]
		notes = append(notes, noteDebugInfo{
			Path:      nv.Path,
			VersionID: nv.VersionID,
			PathID:    nv.PathID,
			Title:     nv.Title,
			HasForm:   hasForm || hasForms || hasFormRef,
		})
	}

	data, err := json.Marshal(notes)
	if err != nil {
		a.log.Error("failed to marshal note debug list", "error", err)
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return true
	}

	ctx.SetBody(data)
	return true
}

func (a *app) handleDebugFormSpec(ctx *fasthttp.RequestCtx) bool {
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)

	path := string(ctx.QueryArgs().Peek("path"))
	if path == "" {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(`{"error":"missing path query param"}`)
		return true
	}

	nvs := a.LatestNoteViews()
	nv := nvs.GetByPath(path)
	if nv == nil {
		// try by permalink
		for _, n := range nvs.VisibleList() {
			if n.Permalink == path {
				nv = n
				break
			}
		}
	}
	if nv == nil {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString(`{"error":"note not found"}`)
		return true
	}

	type result struct {
		Path      string                 `json:"path"`
		VersionID int64                  `json:"version_id"`
		RawMeta   map[string]interface{} `json:"raw_meta_keys"`
		FormRaw   interface{}            `json:"form_raw"`
	}

	keys := make(map[string]interface{}, len(nv.RawMeta))
	for k := range nv.RawMeta {
		keys[k] = fmt.Sprintf("%T", nv.RawMeta[k])
	}

	res := result{
		Path:      nv.Path,
		VersionID: nv.VersionID,
		RawMeta:   keys,
		FormRaw:   fmt.Sprintf("%#v", nv.RawMeta["form"]),
	}

	data, _ := json.Marshal(res)
	ctx.SetBody(data)
	return true
}

func (a *app) handleDebugWaitAllJobs(ctx *fasthttp.RequestCtx) bool {
	const maxTimeout = 5 * time.Minute
	pollInterval := 10 * time.Second
	if ms, err := strconv.Atoi(string(ctx.QueryArgs().Peek("interval"))); err == nil && ms > 0 {
		pollInterval = time.Duration(ms) * time.Millisecond
	}

	// Extend write deadline to allow long polling
	err := ctx.Conn().SetWriteDeadline(time.Now().Add(maxTimeout + time.Minute))
	if err != nil {
		a.log.Error("failed to set write deadline", "error", err)
	}

	startTime := time.Now()

	for {
		// Wait first, then check - so recently enqueued jobs have time to be processed
		time.Sleep(pollInterval)

		// Check timeout
		if time.Since(startTime) > maxTimeout {
			ctx.SetStatusCode(fasthttp.StatusGatewayTimeout)
			ctx.SetBodyString("timeout: jobs still pending after 5 minutes")
			return true
		}

		// Get all queue stats
		stats, statsErr := a.Queries.ListGoqiteAllQueueStats(a.ctx)
		if statsErr != nil {
			a.log.Error("failed to get queue stats", "error", statsErr)
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			ctx.SetBodyString("failed to get queue stats: " + statsErr.Error())
			return true
		}

		// Fail only on dead jobs: messages that exhausted MaxReceive and whose
		// timeout expired — goqite will never deliver them again, so the queue
		// can never drain. A retried job (received > 1) that still has
		// attempts left is the system recovering from a transient error; keep
		// waiting for it instead.
		for _, stat := range stats {
			if stat.RetryCount == 0 {
				continue
			}

			if msg, dead := a.describeDeadJobs(stat.Queue); dead {
				ctx.SetStatusCode(fasthttp.StatusInternalServerError)
				ctx.SetBodyString(msg)
				return true
			}
		}

		// Check if any jobs exist
		totalJobs := int64(0)
		for _, stat := range stats {
			totalJobs += stat.TotalJobs
		}

		if totalJobs == 0 {
			ctx.SetStatusCode(fasthttp.StatusOK)
			ctx.SetBodyString("ok: all jobs completed")
			return true
		}

		a.log.Debug("waiting for jobs to complete", "total_jobs", totalJobs)
	}
}

// isDeadJob reports whether a goqite message can never be delivered again:
// it used up all maxReceive attempts and its visibility timeout expired. An
// unparseable timeout counts as expired.
func isDeadJob(received int64, maxReceive int, timeout string, now time.Time) bool {
	if received < int64(maxReceive) {
		return false
	}

	t, err := time.Parse(time.RFC3339, timeout)
	if err != nil {
		return true
	}

	return !t.After(now)
}

// describeDeadJobs reports whether the queue holds dead jobs (exhausted
// MaxReceive, timed out) and builds a diagnostic message listing them with
// their payloads, so test failures name the exact stuck job instead of just
// the queue.
func (a *app) describeDeadJobs(queue string) (string, bool) {
	maxReceive := 3
	if aq, ok := a.appQueues[queue]; ok {
		maxReceive = aq.maxReceive
	}

	jobs, err := a.Queries.ListGoqiteJobsByQueue(a.ctx, db.ListGoqiteJobsByQueueParams{
		Queue: queue,
		Limit: 50,
	})
	if err != nil {
		return "queue " + queue + ": failed to list jobs: " + err.Error(), true
	}

	now := time.Now().UTC()
	msg := "queue " + queue + " has dead jobs (exhausted retries)"
	dead := false

	for _, job := range jobs {
		if !isDeadJob(job.Received, maxReceive, job.Timeout, now) {
			continue
		}

		dead = true

		body := string(job.Body)
		if len(body) > 300 {
			body = body[:300] + "..."
		}

		msg += fmt.Sprintf("\n  id=%s received=%d created=%s body=%s", job.ID, job.Received, job.Created, body)
	}

	return msg, dead
}

func (a *app) handleDebugRunCronJob(ctx *fasthttp.RequestCtx) bool {
	name := string(ctx.QueryArgs().Peek("name"))
	if name == "" {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString("missing 'name' query parameter")
		return true
	}

	// Get all cron jobs and find by name
	jobs, err := a.Queries.ListAllCronJobs(a.ctx)
	if err != nil {
		a.log.Error("failed to list cron jobs", "error", err)
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString("failed to list cron jobs: " + err.Error())
		return true
	}

	// Build map and find job
	var jobID int64 = -1
	for _, job := range jobs {
		if job.Name == name {
			jobID = job.ID
			break
		}
	}

	if jobID == -1 {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString("cron job not found: " + name)
		return true
	}

	// Execute the job
	execution, err := a.CronJobs.ExecuteCronJobManually(jobID)
	if err != nil {
		a.log.Error("failed to run cron job", "name", name, "error", err)
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString("failed to run cron job: " + err.Error())
		return true
	}

	// Return execution result
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)

	data, err := json.Marshal(execution)
	if err != nil {
		a.log.Error("failed to marshal execution", "error", err)
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString("failed to marshal response")
		return true
	}

	ctx.SetBody(data)
	return true
}

func (a *app) handleDebugTelegramImport(ctx *fasthttp.RequestCtx) bool {
	chatIDStr := string(ctx.QueryArgs().Peek("chat_id"))
	if chatIDStr == "" {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString("missing 'chat_id' query parameter")
		return true
	}

	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString("invalid 'chat_id' query parameter: " + err.Error())
		return true
	}

	// Get first telegram account
	accounts, err := a.Queries.ListAllTelegramAccounts(a.ctx)
	if err != nil {
		a.log.Error("failed to list telegram accounts", "error", err)
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString("failed to list telegram accounts: " + err.Error())
		return true
	}

	if len(accounts) == 0 {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString("no telegram accounts found")
		return true
	}

	params := model.ImportTelegramChannelParams{
		AccountID: accounts[0].ID,
		ChannelID: chatID,
		BasePath:  "import",
		WithMedia: true,
	}

	err = importtelegramchannel.Resolve(a.ctx, a, params)
	if err != nil {
		a.log.Error("failed to import telegram channel", "error", err)
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString("failed to import telegram channel: " + err.Error())
		return true
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyString("ok: telegram channel imported")
	return true
}

func (a *app) handleDebugSpawnJobs(ctx *fasthttp.RequestCtx) bool {
	count, _ := strconv.Atoi(string(ctx.QueryArgs().Peek("count")))
	if count <= 0 {
		count = 1
	}
	durationMs, _ := strconv.Atoi(string(ctx.QueryArgs().Peek("durationMs")))
	tag := string(ctx.QueryArgs().Peek("tag"))

	for i := range count {
		data, _ := json.Marshal(debugSleepJobParams{Tag: tag, DurationMs: durationMs})
		err := a.EnqueueJob(a.ctx, model.BackgroundTask{
			ID:    debugSleepJobID,
			Queue: model.BackgroundDefaultQueue,
			Data:  json.RawMessage(data),
		})
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			ctx.SetBodyString(fmt.Sprintf("failed to enqueue job %d: %v", i, err))
			return true
		}
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyString(fmt.Sprintf("ok: enqueued %d jobs", count))
	return true
}

func (a *app) handleDebugCompletedJobs(ctx *fasthttp.RequestCtx) bool {
	a.debugJobMu.Lock()
	log := make([]debugJobRecord, len(a.debugJobLog))
	copy(log, a.debugJobLog)
	a.debugJobMu.Unlock()

	data, err := json.Marshal(log)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString("failed to marshal log: " + err.Error())
		return true
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody(data)
	return true
}

func (a *app) handleDebugCompletedJobsClear(ctx *fasthttp.RequestCtx) bool {
	a.debugJobMu.Lock()
	a.debugJobLog = nil
	a.debugJobMu.Unlock()

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyString("ok: completed jobs log cleared")
	return true
}

func (a *app) handleDebugTestWebhook(ctx *fasthttp.RequestCtx) bool {
	// Parse query params.
	statusCode := 200
	if s := string(ctx.QueryArgs().Peek("status")); s != "" {
		parsed, parseErr := strconv.Atoi(s)
		if parseErr == nil {
			statusCode = parsed
		}
	}

	delayStr := string(ctx.QueryArgs().Peek("delay"))
	if delayStr != "" {
		delay, parseErr := time.ParseDuration(delayStr)
		if parseErr == nil && delay > 0 {
			time.Sleep(delay)
		}
	}

	// Save the call.
	headers := make(map[string]string)
	//nolint:staticcheck // VisitAll is the correct API for fasthttp.
	ctx.Request.Header.VisitAll(func(key, value []byte) {
		headers[string(key)] = string(value)
	})

	body := ctx.Request.Body()
	var rawBody json.RawMessage
	if json.Valid(body) {
		rawBody = make(json.RawMessage, len(body))
		copy(rawBody, body)
	} else {
		marshaledStr, _ := json.Marshal(string(body))
		rawBody = json.RawMessage(marshaledStr)
	}

	call := webhookTestCall{
		Timestamp: time.Now().Unix(),
		Headers:   headers,
		Body:      rawBody,
	}

	a.webhookTestMu.Lock()
	a.webhookTestCalls = append(a.webhookTestCalls, call)
	a.webhookTestMu.Unlock()

	// Respond.
	ctx.SetStatusCode(statusCode)
	ctx.SetContentType("application/json")

	responseBody := string(ctx.QueryArgs().Peek("body"))
	if responseBody != "" {
		ctx.SetBodyString(responseBody)
	} else {
		// Echo mode: return received body.
		ctx.SetBody(body)
	}

	return true
}

func (a *app) handleDebugTestWebhookCalls(ctx *fasthttp.RequestCtx) bool {
	a.webhookTestMu.Lock()
	calls := make([]webhookTestCall, len(a.webhookTestCalls))
	copy(calls, a.webhookTestCalls)
	a.webhookTestMu.Unlock()

	// Check if only last call requested.
	if string(ctx.QueryArgs().Peek("last")) == "1" && len(calls) > 0 {
		calls = calls[len(calls)-1:]
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)

	data, err := json.Marshal(calls)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString("failed to marshal calls: " + err.Error())
		return true
	}

	ctx.SetBody(data)
	return true
}

func (a *app) handleDebugTestWebhookCallsClear(ctx *fasthttp.RequestCtx) bool {
	a.webhookTestMu.Lock()
	a.webhookTestCalls = nil
	a.webhookTestMu.Unlock()

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyString("ok: webhook calls cleared")
	return true
}
