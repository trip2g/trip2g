package backjobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
	"trip2g/internal/backjobs/sendsignincode"
	"trip2g/internal/logger"

	"maragu.dev/goqite"
	"maragu.dev/goqite/jobs"
)

type Env interface {
	Logger() logger.Logger
	DBConnection() *sql.DB

	// don't forget to embed all env interfaces here
	sendsignincode.Env
}

type BackJobs struct {
	env Env
	log logger.Logger

	queue  *goqite.Queue
	runner *jobs.Runner
}

func New(env Env) *BackJobs {
	log := logger.WithPrefix(env.Logger(), "backjobs:")

	tasks := BackJobs{
		env: env,
		log: log,
	}

	tasks.queue = goqite.New(goqite.NewOpts{
		DB:   env.DBConnection(),
		Name: "back_jobs",
	})

	tasks.runner = jobs.NewRunner(jobs.NewRunnerOpts{
		Limit:        2,
		Log:          logger.WithPrefix(log, "runner:"),
		PollInterval: time.Second,
		Queue:        tasks.queue,
	})

	registerJob(&tasks, sendsignincode.ID, sendsignincode.Resolve)

	return &tasks
}

func (t *BackJobs) queueJob(ctx context.Context, id string, params interface{}) error {
	data, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal params: %w", err)
	}

	return jobs.Create(ctx, t.queue, id, data)
}

func (t *BackJobs) QueueRequestSignInEmail(ctx context.Context, email string, code string) error {
	return t.queueJob(ctx, sendsignincode.ID, sendsignincode.Params{Email: email, Code: code})
}

// registerJob is a generic helper function to register background jobs.
// T is the parameter type for the job.
func registerJob[T any, P any](tasks *BackJobs, jobID string, resolveFunc func(context.Context, T, P) error) {
	tasks.runner.Register(jobID, func(ctx context.Context, m []byte) error {
		var params P

		err := json.Unmarshal(m, &params)
		if err != nil {
			return fmt.Errorf("failed to unmarshal %s params: %w", jobID, err)
		}

		err = resolveFunc(ctx, tasks.env.(T), params) //nolint:errcheck // backjobs.Env should embeded all env interfaces
		if err != nil {
			return fmt.Errorf("failed to run %s job: %w", jobID, err)
		}

		return nil
	})
}
