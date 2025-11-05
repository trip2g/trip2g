package updatetelegrampost

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/telegram"
)

const JobID = "update_message"
const QueueID = model.BackgroundTelegramAPICallQueue
const Priority = 0

type UpdateTelegramPostEnv interface {
	Env
	Logger() logger.Logger
	RegisterJob(qID model.BackgroundQueueID, id string, handler func(ctx context.Context, m []byte) error)
	EnqueueJob(ctx context.Context, job model.BackgroundTask) error
}

type UpdateTelegramPostJob struct {
	env     UpdateTelegramPostEnv
	enqueue func(ctx context.Context, params model.TelegramUpdatePostParams) error
}

func New(env UpdateTelegramPostEnv) *UpdateTelegramPostJob {
	task := &UpdateTelegramPostJob{env: env}

	jobTimeout := time.Minute

	// Register the job handler with custom logic for rate limiting
	env.RegisterJob(QueueID, JobID, func(ctx context.Context, m []byte) error {
		var params model.TelegramUpdatePostParams

		err := json.Unmarshal(m, &params)
		if err != nil {
			return fmt.Errorf("failed to unmarshal update_telegram_post params: %w", err)
		}

		// independent timeout context from stop cancelations
		jobCtx, cancel := context.WithTimeout(context.Background(), jobTimeout)
		defer cancel()

		err = Resolve(jobCtx, env, params)
		if err != nil {
			shouldRetry, delay := telegram.HandleRateLimit(err)
			if shouldRetry {
				env.Logger().Info("telegram rate limit hit, retrying after delay",
					"delay", delay,
					"job", JobID,
				)
				time.Sleep(delay)
				err = Resolve(ctx, env, params)
			}

			if err != nil {
				return err
			}
		}

		return nil
	})

	// Create the enqueue function
	task.enqueue = func(ctx context.Context, params model.TelegramUpdatePostParams) error {
		return env.EnqueueJob(ctx, model.BackgroundTask{
			ID:       JobID,
			Queue:    QueueID,
			Data:     params,
			Priority: Priority,
		})
	}

	return task
}

func (t UpdateTelegramPostJob) EnqueueUpdateTelegramPost(ctx context.Context, params model.TelegramUpdatePostParams) error {
	return t.enqueue(ctx, params)
}
