package main

import (
	"trip2g/internal/case/cronjob/cleanupapikeylogs"
	"trip2g/internal/case/cronjob/cleanupwebhookdeliveries"
	"trip2g/internal/case/cronjob/cleanupwebhookdeliverylogs"
	"trip2g/internal/case/cronjob/clearcronjobexecutionhistory"
	"trip2g/internal/case/cronjob/executecronwebhooks"
	"trip2g/internal/case/cronjob/expirestalewebhookdeliveries"
	"trip2g/internal/case/cronjob/materializegitmirror"
	"trip2g/internal/case/cronjob/refreshtelegramaccounts"
	"trip2g/internal/case/cronjob/refreshtelegramchatusernames"
	"trip2g/internal/case/cronjob/regeneratenoteembeddings"
	"trip2g/internal/case/cronjob/removeexpiredtgchatmembers"
	"trip2g/internal/case/cronjob/sendscheduledtelegrampublishposts"
	"trip2g/internal/case/cronjob/simplebackup"
	"trip2g/internal/case/cronjob/updatetelegrampublishposts"
	"trip2g/internal/case/cronjob/vacuumdatabase"
	"trip2g/internal/cronjobs"
)

func getCronJobConfigs(app *app) []cronjobs.Job {
	// Each New(app, ...) captures the typed env at wiring time — the constructor
	// call is the compile-time proof that *app implements the job's Env.
	jobs := []cronjobs.Job{
		materializegitmirror.New(app),
		removeexpiredtgchatmembers.New(app),
		clearcronjobexecutionhistory.New(app),
		sendscheduledtelegrampublishposts.New(app, app.config.CronTelegramPublishSchedule),
		updatetelegrampublishposts.New(app),
		refreshtelegramaccounts.New(app),
		refreshtelegramchatusernames.New(app),
		regeneratenoteembeddings.New(app),
		executecronwebhooks.New(app, app.config.CronExecuteWebhooksSchedule),
		cleanupwebhookdeliverylogs.New(app, app.config.WebhookDeliveryLogs),
		cleanupwebhookdeliveries.New(app, app.config.WebhookDeliveries),
		cleanupapikeylogs.New(app, app.config.APIKeyLogs),
		expirestalewebhookdeliveries.New(app),
	}

	// VACUUM/ANALYZE maintenance is opt-in (heavy full-DB rewrite; incompatible
	// with Litestream, which owns WAL checkpointing).
	if app.config.VacuumCron {
		jobs = append(jobs, vacuumdatabase.New(app))
	}

	// Conditionally add simple backup job if enabled
	if app.simpleBackup != nil {
		jobs = append(jobs, simplebackup.New(app))
	}

	return jobs
}
