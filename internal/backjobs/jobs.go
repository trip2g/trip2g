package backjobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"trip2g/internal/backjobs/sendsignincode"
	"trip2g/internal/logger"
)

type Env interface {
	Logger() logger.Logger
	EnqueueJob(ctx context.Context, jobID string, data []byte) error
	RegisterJob(id string, handler func(ctx context.Context, m []byte) error)

	// don't forget to embed all env interfaces here
	sendsignincode.Env
}

type RequestEnv interface {
	CurrentTx() *sql.Tx
}

type BackJobs struct {
	env Env
	log logger.Logger
}

func New(env Env) *BackJobs {
	log := logger.WithPrefix(env.Logger(), "backjobs:")

	tasks := BackJobs{
		env: env,
		log: log,
	}

	// Register jobs with backjobs prefix
	registerJob(&tasks, sendsignincode.ID, sendsignincode.Resolve)

	return &tasks
}

func (t *BackJobs) queueJob(ctx context.Context, jobID string, params interface{}) error {
	data, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal params: %w", err)
	}

	return t.env.EnqueueJob(ctx, jobQueueID(jobID), data)
}

func jobQueueID(id string) string {
	return fmt.Sprintf("backjobs:%s", id)
}

func (t *BackJobs) QueueRequestSignInEmail(ctx context.Context, email string, code string) error {
	return t.queueJob(ctx, sendsignincode.ID, sendsignincode.Params{Email: email, Code: code})
}

// registerJob is a generic helper function to register background jobs.
// T is the parameter type for the job.
func registerJob[T any, P any](tasks *BackJobs, jobID string, resolveFunc func(context.Context, T, P) error) {
	id := jobQueueID(jobID)

	tasks.env.RegisterJob(id, func(ctx context.Context, m []byte) error {
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
