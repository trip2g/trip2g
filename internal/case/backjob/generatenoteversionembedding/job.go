package generatenoteversionembedding

import (
	"context"
	"errors"
	"fmt"
	"time"

	"trip2g/internal/jobs"
	"trip2g/internal/model"
)

const JobID = "generate_note_version_embedding"
const QueueID = model.BackgroundDefaultQueue
const Priority = 10 // Lower priority than user-facing jobs

// MaxAttempts bounds how many times one note version is tried. After the last
// failure the version waits for the regenerate_note_embeddings sweep.
const MaxAttempts = 10

const (
	firstRetryDelay = time.Minute
	maxRetryDelay   = time.Hour
)

type Job struct {
	enqueue jobs.EnqueueFunc
}

type JobEnv interface {
	jobs.Env
	Env
}

func New(env JobEnv) *Job {
	return &Job{
		enqueue: jobs.Register(env, QueueID, JobID, Priority,
			func(ctx context.Context, params Params) error {
				return Run(ctx, env, params)
			}),
	}
}

func (j *Job) Enqueue(ctx context.Context, versionID int64) error {
	return j.enqueue(ctx, Params{VersionID: versionID})
}

// Run embeds one note version. A failure is retried by a new delayed job
// instead of the queue's immediate redelivery: a slow endpoint only falls
// further behind when it is retried at once, and the queue gives a message up
// for good after a few receives. The failed job is finished, so a version
// never has more than one retry waiting.
func Run(ctx context.Context, env JobEnv, params Params) error {
	err := Resolve(ctx, env, params)
	if err == nil || params.Attempt+1 >= MaxAttempts {
		return err
	}

	retry := model.BackgroundTask{
		ID:       JobID,
		Queue:    QueueID,
		Data:     Params{VersionID: params.VersionID, Attempt: params.Attempt + 1},
		Priority: Priority,
		Delay:    RetryDelay(params.Attempt),
	}

	enqueueErr := env.EnqueueJob(ctx, retry)
	if enqueueErr != nil {
		return errors.Join(err, fmt.Errorf("failed to schedule retry: %w", enqueueErr))
	}

	env.Logger().Warn("embedding failed, retry scheduled",
		"version_id", params.VersionID,
		"attempt", params.Attempt+1,
		"retry_in", retry.Delay,
		"error", err,
	)

	return nil
}

// RetryDelay doubles from firstRetryDelay with each failed attempt, capped at
// maxRetryDelay.
func RetryDelay(attempt int) time.Duration {
	delay := firstRetryDelay
	for range attempt {
		delay *= 2
		if delay >= maxRetryDelay {
			return maxRetryDelay
		}
	}
	return delay
}
