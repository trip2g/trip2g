package cronjobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"
	"trip2g/internal/db"
	"trip2g/internal/logger"

	"github.com/robfig/cron/v3"
	"maragu.dev/goqite"
	"maragu.dev/goqite/jobs"
)

var (
	JobStatusPending   int64 = 0
	JobStatusRunning   int64 = 1
	JobStatusCompleted int64 = 2
	JobStatusFailed    int64 = 3
)

const executeJobName = "e"

type Job interface {
	Name() string
	Schedule() string
	ExecuteAfterStart() bool
	Execute(ctx context.Context, env interface{}) (interface{}, error)
}

type Env interface {
	UpsertCronJob(ctx context.Context, arg db.UpsertCronJobParams) error
	InsertCronJobExecution(ctx context.Context, jobID int64) (db.CronJobExecution, error)
	UpdateCronJobExecution(ctx context.Context, arg db.UpdateCronJobExecutionParams) error
	UpdateCronJobLastExec(ctx context.Context, id int64) error
	UpdateRunningCronJobExecutions(ctx context.Context, params db.UpdateRunningCronJobExecutionsParams) error
	CronJobByName(ctx context.Context, name string) (db.CronJob, error)
	Logger() logger.Logger
	DBConnection() *sql.DB
}

type jobItem struct {
	job    Job
	config db.CronJob
	cronID cron.EntryID
	queue  *goqite.Queue
	runner *jobs.Runner
}

type CronJobs struct {
	env  Env
	ctx  context.Context
	cron *cron.Cron

	log logger.Logger

	mu   sync.Mutex
	jobs map[int64]*jobItem

	queues  map[int64]*goqite.Queue
	runners map[int64]*jobs.Runner
}

func New(ctx context.Context, env Env, jobConfigs []Job) (*CronJobs, error) {
	cj := &CronJobs{
		ctx:     ctx,
		env:     env,
		cron:    cron.New(cron.WithSeconds()),
		runners: make(map[int64]*jobs.Runner),
		log:     logger.WithPrefix(env.Logger(), "cronjobs:"),

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

		queue := goqite.New(goqite.NewOpts{
			DB:   env.DBConnection(),
			Name: name,
		})

		runner := jobs.NewRunner(jobs.NewRunnerOpts{
			Limit:        1, // limit to one job at a time
			Log:          logger.WithPrefix(cj.log, fmt.Sprintf("jobs:%s:", name)),
			PollInterval: time.Second,
			Queue:        queue,
		})

		runner.Register(executeJobName, func(ctx context.Context, m []byte) error {
			return cj.executeJob(dbJob.ID)
		})

		cj.jobs[dbJob.ID] = &jobItem{
			job:    job,
			config: dbJob,
			queue:  queue,
			runner: runner,
		}

		go runner.Start(ctx)

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
	q := cj.queues[jobID]

	err := jobs.Create(cj.ctx, q, executeJobName, nil)
	if err != nil {
		cj.log.Error("failed to create test job", "job_id", jobID, "error", err)
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

func (cj *CronJobs) executeJob(jobID int64) error {
	job, ok := cj.jobs[jobID]
	if !ok {
		return fmt.Errorf("job %s not found", jobID)
	}

	err := cj.env.UpdateRunningCronJobExecutions(cj.ctx, db.UpdateRunningCronJobExecutionsParams{
		JobID:  jobID,
		Status: JobStatusRunning,
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

func (cj *CronJobs) RefreshCronJob(job db.CronJob) error {
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

func (cj *CronJobs) ExecuteCronJobJobManually(jobID int64) error {
	if _, ok := cj.jobs[jobID]; !ok {
		return fmt.Errorf("job %d not found", jobID)
	}

	return cj.executeJob(jobID)
}

func (cj *CronJobs) StopCronJobs() {
	cj.cron.Stop()
}
