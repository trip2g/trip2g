package main

import (
	"encoding/json"
	"strings"

	"github.com/valyala/fasthttp"
)

func (a *app) handleDebugAPI(ctx *fasthttp.RequestCtx) bool {
	if !a.config.DevMode {
		return false
	}

	path := string(ctx.Path())

	switch {
	case strings.HasPrefix(path, "/debug/layouts/latest"):
		return a.handleDebugLayoutsLatest(ctx)

	case strings.HasPrefix(path, "/debug/nvs/latest"):
		return a.handleDebugNvsLatest(ctx)

	case strings.HasPrefix(path, "/debug/run_cron_job"):
		return a.handleDebugRunCronJob(ctx)
	}

	return false
}

func (a *app) handleDebugLayoutsLatest(ctx *fasthttp.RequestCtx) bool {
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)

	data, err := json.Marshal(a.Layouts()) //nolint:musttag // debug endpoint
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
