package main

import (
	"trip2g/internal/case/cronjob/applygitchanges"
	"trip2g/internal/case/cronjob/clearcronjobexecutionhistory"
	"trip2g/internal/case/cronjob/cleanupwebhookdeliveries"
	"trip2g/internal/case/cronjob/cleanupwebhookdeliverylogs"
	"trip2g/internal/case/cronjob/executecronwebhooks"
	"trip2g/internal/case/cronjob/refreshtelegramaccounts"
	"trip2g/internal/case/cronjob/regeneratenoteembeddings"
	"trip2g/internal/case/cronjob/removeexpiredtgchatmembers"
	"trip2g/internal/case/cronjob/sendscheduledtelegrampublishposts"
	"trip2g/internal/case/cronjob/simplebackup"
	"trip2g/internal/case/cronjob/updatetelegrampublishposts"
	"trip2g/internal/case/cronjob/vacuumdatabase"
	"trip2g/internal/cronjobs"
)

func getCronJobConfigs(app *app) []cronjobs.Job {
	// Compile-time interface checks
	var (
		_ simplebackup.Env   = app
		_ vacuumdatabase.Env = app

		_ applygitchanges.Env              = app
		_ removeexpiredtgchatmembers.Env   = app
		_ clearcronjobexecutionhistory.Env = app

		_ sendscheduledtelegrampublishposts.Env = app
		_ updatetelegrampublishposts.Env        = app
		_ refreshtelegramaccounts.Env           = app

		_ regeneratenoteembeddings.Env = app

		_ executecronwebhooks.Env = app

		_ cleanupwebhookdeliverylogs.Env = app
		_ cleanupwebhookdeliveries.Env   = app
	)

	jobs := []cronjobs.Job{
		&applygitchanges.Job{},
		&removeexpiredtgchatmembers.Job{},
		&clearcronjobexecutionhistory.Job{},
		&sendscheduledtelegrampublishposts.Job{},
		&updatetelegrampublishposts.Job{},
		&refreshtelegramaccounts.Job{},
		&vacuumdatabase.Job{},
		&regeneratenoteembeddings.Job{},
		&executecronwebhooks.Job{},
		&cleanupwebhookdeliverylogs.Job{},
		&cleanupwebhookdeliveries.Job{},
	}

	// Conditionally add simple backup job if enabled
	if app.simpleBackup != nil {
		jobs = append(jobs, &simplebackup.Job{})
	}

	return jobs
}
