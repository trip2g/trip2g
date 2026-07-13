package main

import (
	"trip2g/internal/case/cronjob/cleanupapikeylogs"
	"trip2g/internal/case/cronjob/cleanupwebhookdeliveries"
	"trip2g/internal/case/cronjob/cleanupwebhookdeliverylogs"
	"trip2g/internal/case/cronjob/clearcronjobexecutionhistory"
	"trip2g/internal/case/cronjob/executecronwebhooks"
	"trip2g/internal/case/cronjob/expirestalewebhookdeliveries"
	"trip2g/internal/case/cronjob/refreshtelegramaccounts"
	"trip2g/internal/case/cronjob/refreshtelegramchatusernames"
	"trip2g/internal/case/cronjob/regeneratenoteembeddings"
	"trip2g/internal/case/cronjob/removeexpiredtgchatmembers"
	"trip2g/internal/case/cronjob/sendscheduledtelegrampublishposts"
	"trip2g/internal/case/cronjob/simplebackup"
	"trip2g/internal/case/cronjob/materializegitmirror"
	"trip2g/internal/case/cronjob/updatetelegrampublishposts"
	"trip2g/internal/case/cronjob/vacuumdatabase"
	"trip2g/internal/cronjobs"
)

func getCronJobConfigs(app *app) []cronjobs.Job {
	// Compile-time interface checks
	var (
		_ simplebackup.Env   = app
		_ vacuumdatabase.Env = app

		_ materializegitmirror.Env         = app
		_ removeexpiredtgchatmembers.Env   = app
		_ clearcronjobexecutionhistory.Env = app

		_ sendscheduledtelegrampublishposts.Env = app
		_ updatetelegrampublishposts.Env        = app
		_ refreshtelegramaccounts.Env           = app
		_ refreshtelegramchatusernames.Env      = app

		_ regeneratenoteembeddings.Env = app

		_ executecronwebhooks.Env = app

		_ cleanupwebhookdeliverylogs.Env   = app
		_ cleanupwebhookdeliveries.Env     = app
		_ cleanupapikeylogs.Env            = app
		_ expirestalewebhookdeliveries.Env = app
	)

	jobs := []cronjobs.Job{
		&materializegitmirror.Job{},
		&removeexpiredtgchatmembers.Job{},
		&clearcronjobexecutionhistory.Job{},
		&sendscheduledtelegrampublishposts.Job{Cron: app.config.CronTelegramPublishSchedule},
		&updatetelegrampublishposts.Job{},
		&refreshtelegramaccounts.Job{},
		&refreshtelegramchatusernames.Job{},
		&regeneratenoteembeddings.Job{},
		&executecronwebhooks.Job{Cron: app.config.CronExecuteWebhooksSchedule},
		&cleanupwebhookdeliverylogs.Job{},
		&cleanupwebhookdeliveries.Job{},
		&cleanupapikeylogs.Job{Config: app.config.APIKeyLogs},
		&expirestalewebhookdeliveries.Job{},
	}

	// VACUUM/ANALYZE maintenance is opt-in (heavy full-DB rewrite; incompatible
	// with Litestream, which owns WAL checkpointing).
	if app.config.VacuumCron {
		jobs = append(jobs, &vacuumdatabase.Job{})
	}

	// Conditionally add simple backup job if enabled
	if app.simpleBackup != nil {
		jobs = append(jobs, &simplebackup.Job{})
	}

	return jobs
}
