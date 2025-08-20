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
	"maragu.dev/goqite"
	"maragu.dev/goqite/jobs"
)

const (
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
	UpsertCronJob(ctx context.Context, arg db.UpsertCronJobParams) error
	InsertCronJobExecution(ctx context.Context, jobID int64) (db.CronJobExecution, error)
	UpdateCronJobExecution(ctx context.Context, arg db.UpdateCronJobExecutionParams) (db.CronJobExecution, error)
	UpdateCronJobLastExec(ctx context.Context, id int64) error
	UpdateRunningCronJobExecutions(ctx context.Context, params db.UpdateRunningCronJobExecutionsParams) error
	CronJobByName(ctx context.Context, name string) (db.CronJob, error)
	Logger() logger.Logger

	GoqiteQueue() *goqite.Queue
	GoqiteRunner() *jobs.Runner
}

type jobItem struct {
	job    Job
	config db.CronJob
	cronID cron.EntryID
}

type CronJobs struct {
	env  Env
	ctx  context.Context
	cron *cron.Cron

	log logger.Logger

	mu   sync.Mutex
	jobs map[int64]*jobItem
}

func New(ctx context.Context, env Env, jobConfigs []Job) (*CronJobs, error) {
	cj := &CronJobs{
		ctx:  ctx,
		env:  env,
		cron: cron.New(cron.WithSeconds()),
		log:  logger.WithPrefix(env.Logger(), "cronjobs:"),

		mu:   sync.Mutex{},
		jobs: make(map[int64]*jobItem),
	}

	// Register all jobs
	for _, job := range jobConfigs {
		name := job.Name()

		upsertParams := db.UpsertCronJobParams{
			Name:       name,
			Expression: job.Schedule(),
		}

		err := env.UpsertCronJob(ctx, upsertParams)
		if err != nil {
			return nil, fmt.Errorf("failed to upsert cron job %s: %w", name, err)
		}

		// get current cron job settings
		dbJob, err := env.CronJobByName(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get cron job %s from database: %w", name, err)
		}

		// Register job with cronjobs prefix
		jobName := fmt.Sprintf("cronjobs:%s", name)
		cj.env.GoqiteRunner().Register(jobName, func(ctx context.Context, m []byte) error {
			_, execErr := cj.executeJob(dbJob.ID)
			return execErr
		})

		cj.jobs[dbJob.ID] = &jobItem{
			job:    job,
			config: dbJob,
		}

		if dbJob.Enabled {
			err = cj.register(dbJob.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to register cron job %s: %w", dbJob.Name, err)
			}
		}
	}

	// Start the cron scheduler
	cj.cron.Start()

	return cj, nil
}

func (cj *CronJobs) scheduleJob(jobID int64) {
	job, ok := cj.jobs[jobID]
	if !ok {
		cj.log.Error("job not found for scheduling", "job_id", jobID)
		return
	}

	jobName := fmt.Sprintf("cronjobs:%s", job.job.Name())
	err := jobs.Create(cj.ctx, cj.env.GoqiteQueue(), jobName, nil)
	if err != nil {
		cj.log.Error("failed to create job", "job_id", jobID, "job_name", jobName, "error", err)
	}
}

func (cj *CronJobs) register(jobID int64) error {
	cj.mu.Lock()
	defer cj.mu.Unlock()

	job := cj.jobs[jobID]

	entryID, err := cj.cron.AddFunc(job.config.Expression, func() {
		cj.scheduleJob(jobID)
	})
	if err != nil {
		return fmt.Errorf("failed to AddFunc %s: %w", job.job.Name(), err)
	}

	cj.jobs[jobID].cronID = entryID

	return nil
}

func (cj *CronJobs) executeJob(jobID int64) (*db.CronJobExecution, error) {
	job, ok := cj.jobs[jobID]
	if !ok {
		return nil, fmt.Errorf("job %d not found", jobID)
	}

	updateErr := cj.env.UpdateRunningCronJobExecutions(cj.ctx, db.UpdateRunningCronJobExecutionsParams{
		JobID:  jobID,
		Status: JobStatusRunning,
		ErrorMessage: sql.NullString{
			Valid:  true,
			String: "died",
		},
	})
	if updateErr != nil {
		cj.log.Error("failed to update running cron job executions", "job_id", jobID, "error", updateErr)
	}

	// Insert execution record
	exec, err := cj.env.InsertCronJobExecution(cj.ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to insert cron job execution for %d: %w", jobID, err)
	}

	// Update status to running
	_, err = cj.env.UpdateCronJobExecution(cj.ctx, db.UpdateCronJobExecutionParams{
		ID:     exec.ID,
		Status: JobStatusRunning,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update cron job execution status for %d: %w", jobID, err)
	}

	// Execute the job
	report, jobErr := job.job.Execute(cj.ctx, cj.env)

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
		reportDataRaw, marshalErr := json.Marshal(report)
		if marshalErr != nil {
			return nil, fmt.Errorf("failed to marshal report data for job %d: %w", jobID, marshalErr)
		}

		reportData.String = string(reportDataRaw)
		reportData.Valid = true
	}

	execution, err := cj.env.UpdateCronJobExecution(cj.ctx, db.UpdateCronJobExecutionParams{
		ID:           exec.ID,
		Status:       status,
		ErrorMessage: errorMessage,
		ReportData:   reportData,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update cron job execution for %d: %w", jobID, err)
	}

	// Update last execution time
	err = cj.env.UpdateCronJobLastExec(cj.ctx, exec.JobID)
	if err != nil {
		return nil, fmt.Errorf("failed to update last execution time for cron job %d: %w", jobID, err)
	}

	return &execution, nil
}

func (cj *CronJobs) RefreshCronJob(job db.CronJob) error {
	cj.log.Info("refreshing", "job_id", job.ID, "name", job.Name)

	// Remove existing entry if exists
	jobItem, exists := cj.jobs[job.ID]
	if !exists {
		return fmt.Errorf("job %s not found", job.Name)
	}

	cj.cron.Remove(jobItem.cronID)

	// Register again if enabled
	if job.Enabled {
		return cj.register(job.ID)
	}

	return nil
}

func (cj *CronJobs) ExecuteCronJobJobManually(jobID int64) (*db.CronJobExecution, error) {
	if _, ok := cj.jobs[jobID]; !ok {
		return nil, fmt.Errorf("job %d not found", jobID)
	}

	return cj.executeJob(jobID)
}

func (cj *CronJobs) StopCronJobs() {
	cj.cron.Stop()
}

func (cj *CronJobs) CheckCronjobExpression(val string) bool {
	id, err := cj.cron.AddFunc(val, func() {})
	if err != nil {
		cj.log.Error("failed to test cron job expression", "expression", val, "error", err)
		return false
	}

	cj.cron.Remove(id)

	return true
}
