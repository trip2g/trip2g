package sendtelegrampost

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/telegram"
)

const JobID = "send_message"
const QueueID = model.BackgroundTelegramAPICallQueue
const Priority = 1 // shoud process before updates

type SendTelegramPostEnv interface {
	Env
	Logger() logger.Logger
	RegisterJob(qID model.BackgroundQueueID, id string, handler func(ctx context.Context, m []byte) error)
	EnqueueJob(ctx context.Context, job model.BackgroundTask) error
}

type SendTelegramPostJob struct {
	env     SendTelegramPostEnv
	enqueue func(ctx context.Context, params model.TelegramSendPostParams) error
}

func New(env SendTelegramPostEnv) *SendTelegramPostJob {
	task := &SendTelegramPostJob{env: env}

	jobTimeout := time.Minute

	// Register the job handler with custom logic for rate limiting
	env.RegisterJob(QueueID, JobID, func(ctx context.Context, m []byte) error {
		var params model.TelegramSendPostParams

		err := json.Unmarshal(m, &params)
		if err != nil {
			return fmt.Errorf("failed to unmarshal send_telegram_post params: %w", err)
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
	task.enqueue = func(ctx context.Context, params model.TelegramSendPostParams) error {
		return env.EnqueueJob(ctx, model.BackgroundTask{
			ID:       JobID,
			Queue:    QueueID,
			Data:     params,
			Priority: Priority,
		})
	}

	return task
}

func (t SendTelegramPostJob) EnqueueSendTelegramPost(ctx context.Context, params model.TelegramSendPostParams) error {
	return t.enqueue(ctx, params)
}
