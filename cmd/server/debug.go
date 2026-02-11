package main

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/valyala/fasthttp"

	"trip2g/internal/case/backjob/importtelegramchannel"
	"trip2g/internal/model"
)

type webhookTestCall struct {
	Timestamp int64             `json:"timestamp"`
	Headers   map[string]string `json:"headers"`
	Body      json.RawMessage   `json:"body"`
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

	case strings.HasPrefix(path, "/debug/nvs/latest"):
		return a.handleDebugNvsLatest(ctx)

	case strings.HasPrefix(path, "/debug/wait_all_jobs"):
		return a.handleDebugWaitAllJobs(ctx)

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

	data, err := json.Marshal(a.Layouts()) //nolint:musttag,staticcheck // debug endpoint, func fields skipped by json
	if err != nil {
		a.log.Error("failed to marshal latest note views", "error", err)
		return true
	}

	ctx.SetBody(data)
	return true
}

func (a *app) handleDebugNvsLatest(ctx *fasthttp.RequestCtx) bool {
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)

	data, err := json.Marshal(a.LatestNoteViews()) //nolint:musttag // debug endpoint
	if err != nil {
		a.log.Error("failed to marshal latest note views", "error", err)
		return true
	}

	ctx.SetBody(data)
	return true
}

func (a *app) handleDebugWaitAllJobs(ctx *fasthttp.RequestCtx) bool {
	const (
		pollInterval = 10 * time.Second
		maxTimeout   = 5 * time.Minute
	)

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

		// Check for retries (received > 1 means job was retried)
		for _, stat := range stats {
			if stat.RetryCount > 0 {
				ctx.SetStatusCode(fasthttp.StatusInternalServerError)
				ctx.SetBodyString("queue " + stat.Queue + " has failed jobs with retries")
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
