package cronjobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"trip2g/internal/db"
	"trip2g/internal/logger"

	"github.com/robfig/cron/v3"
)

var (
	JobStatusPending   int64 = 0
	JobStatusRunning   int64 = 1
	JobStatusCompleted int64 = 2
	JobStatusFailed    int64 = 3
)

type Job interface {
	Name() string
	Schedule() string
	ExecuteAfterStart() bool
	Execute(ctx context.Context, env interface{}) (interface{}, error)
}

type Env interface {
	ListActiveCronJobs(ctx context.Context) ([]db.CronJob, error)
	UpsertCronJob(ctx context.Context, arg db.UpsertCronJobParams) error
	InsertCronJobExecution(ctx context.Context, jobID string) (db.CronJobExecution, error)
	UpdateCronJobExecution(ctx context.Context, arg db.UpdateCronJobExecutionParams) error
	UpdateCronJobLastExec(ctx context.Context, id int64) error
	UpdateRunningCronJobExecutionsByName(ctx context.Context, params db.UpdateRunningCronJobExecutionsByNameParams) error
	Logger() logger.Logger
}

type CronJobs struct {
	env        Env
	ctx        context.Context
	cron       *cron.Cron
	jobs       map[string]Job
	entryIDs   map[string]cron.EntryID
	runningMux sync.RWMutex
	running    map[string]bool
	log        logger.Logger
}

func New(ctx context.Context, env Env, jobConfigs []Job) (*CronJobs, error) {
	cj := &CronJobs{
		ctx:      ctx,
		env:      env,
		cron:     cron.New(cron.WithSeconds()),
		jobs:     make(map[string]Job),
		entryIDs: make(map[string]cron.EntryID),
		running:  make(map[string]bool),
		log:      logger.WithPrefix(env.Logger(), "logger:"),
	}

	// Register all jobs
	for _, job := range jobConfigs {
		name := job.Name()
		cj.jobs[name] = job

		err := env.UpsertCronJob(ctx, db.UpsertCronJobParams{
			Name:       name,
			Expression: job.Schedule(),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to upsert cron job %s: %w", name, err)
		}
	}

	// Load enabled jobs from database
	enabledJobs, err := env.ListActiveCronJobs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list active cron jobs: %w", err)
	}

	for _, dbJob := range enabledJobs {
		if job, ok := cj.jobs[dbJob.Name]; ok {
			err = cj.register(job)
			if err != nil {
				return nil, fmt.Errorf("failed to register cron job %s: %w", dbJob.Name, err)
			}

			if dbJob.Enabled {
				go cj.executeJobWithLog(job.Name())
			}
		}
	}

	// Start the cron scheduler
	cj.cron.Start()

	return cj, nil
}

func (cj *CronJobs) register(job Job) error {
	entryID, err := cj.cron.AddFunc(job.Schedule(), func() {
		cj.executeJobWithLog(job.Name())
	})
	if err != nil {
		return fmt.Errorf("failed to AddFunc %s: %w", job.Name(), err)
	}

	cj.entryIDs[job.Name()] = entryID
	return nil
}

func (cj *CronJobs) executeJobWithLog(jobID string) {
	err := cj.executeJob(jobID)
	if err != nil {
		cj.log.Error("failed to execute cron job", "job_id", jobID, "error", err)
	}
}

func (cj *CronJobs) executeJob(jobID string) error {
	// Check if job is already running
	cj.runningMux.Lock()
	if cj.running[jobID] {
		cj.runningMux.Unlock()
		return nil
	}
	cj.running[jobID] = true
	cj.runningMux.Unlock()

	defer func() {
		cj.runningMux.Lock()
		delete(cj.running, jobID)
		cj.runningMux.Unlock()
	}()

	job, ok := cj.jobs[jobID]
	if !ok {
		return fmt.Errorf("job %s not found", jobID)
	}

	err := cj.env.UpdateRunningCronJobExecutionsByName(cj.ctx, db.UpdateRunningCronJobExecutionsByNameParams{
		Name:   jobID,
		Status: 2,
		ErrorMessage: sql.NullString{
			Valid:  true,
			String: "died",
		},
	})

	// Insert execution record
	exec, err := cj.env.InsertCronJobExecution(cj.ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to insert cron job execution for %s: %w", jobID, err)
	}

	// Update status to running
	err = cj.env.UpdateCronJobExecution(cj.ctx, db.UpdateCronJobExecutionParams{
		ID:     exec.ID,
		Status: JobStatusRunning,
	})
	if err != nil {
		return fmt.Errorf("failed to update cron job execution status for %s: %w", jobID, err)
	}

	// Execute the job
	report, jobErr := job.Execute(cj.ctx, cj.env)

	// Update execution status
	status := JobStatusCompleted

	var (
		errorMessage sql.NullString
		reportData   sql.NullString
	)

	if jobErr != nil {
		status = JobStatusFailed
		errorMessage = sql.NullString{
			String: jobErr.Error(),
			Valid:  true,
		}
	} else {
		reportDataRaw, err := json.Marshal(report)
		if err != nil {
			return fmt.Errorf("failed to marshal report data for job %s: %w", jobID, err)
		}

		reportData.String = string(reportDataRaw)
		reportData.Valid = true
	}

	err = cj.env.UpdateCronJobExecution(cj.ctx, db.UpdateCronJobExecutionParams{
		ID:           exec.ID,
		Status:       status,
		ErrorMessage: errorMessage,
		ReportData:   reportData,
	})
	if err != nil {
		return fmt.Errorf("failed to update cron job execution for %s: %w", jobID, err)
	}

	// Update last execution time
	err = cj.env.UpdateCronJobLastExec(cj.ctx, exec.JobID)
	if err != nil {
		return fmt.Errorf("failed to update last execution time for cron job %s: %w", jobID, err)
	}

	return nil
}

func (cj *CronJobs) RefreshCronJob(dbJob *db.CronJob) error {
	job, ok := cj.jobs[dbJob.Name]
	if !ok {
		return fmt.Errorf("job %s not found", dbJob.ID)
	}

	// Remove existing entry if exists
	if entryID, exists := cj.entryIDs[dbJob.Name]; exists {
		cj.cron.Remove(entryID)
		delete(cj.entryIDs, dbJob.Name)
	}

	// Register again if enabled
	if dbJob.Enabled {
		return cj.register(job)
	}

	return nil
}

func (cj *CronJobs) StopCronJobs() {
	if cj.cron != nil {
		cj.cron.Stop()
	}
}
