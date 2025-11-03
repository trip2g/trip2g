package main

import (
	"trip2g/internal/case/cronjob/applygitchanges"
	"trip2g/internal/case/cronjob/clearcronjobexecutionhistory"
	"trip2g/internal/case/cronjob/removeexpiredtgchatmembers"
	"trip2g/internal/case/cronjob/sendscheduledtelegrampublishposts"
	"trip2g/internal/case/cronjob/updatetelegrampublishposts"
	"trip2g/internal/case/cronjob/vacuumdatabase"
	"trip2g/internal/cronjobs"
)

func getCronJobConfigs(app *app) []cronjobs.Job {
	// Compile-time interface checks
	var (
		_ applygitchanges.Env              = app
		_ removeexpiredtgchatmembers.Env   = app
		_ clearcronjobexecutionhistory.Env = app

		_ sendscheduledtelegrampublishposts.Env = app
		_ updatetelegrampublishposts.Env        = app
		_ vacuumdatabase.Env                    = app
		// _ extractallnotionpages.Env        = app
		// _ otherjob.Env = app
		// _ anotherjob.Env = app
	)

	return []cronjobs.Job{
		&applygitchanges.Job{},
		&removeexpiredtgchatmembers.Job{},
		&clearcronjobexecutionhistory.Job{},
		&sendscheduledtelegrampublishposts.Job{},
		&updatetelegrampublishposts.Job{},
		&vacuumdatabase.Job{},
		// &extractallnotionpages.Job{},
		// &otherjob.Job{},
		// &anotherjob.Job{},
	}
}
