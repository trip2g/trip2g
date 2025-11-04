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

const ID = "update_message"

type UpdateTelegramPostEnv interface {
	Env

	RegisterTelegramJob(id string, handler func(ctx context.Context, m []byte) error)
	EnqueueTelegramJob(ctx context.Context, jobID string, data any, priroity int) error
	Logger() logger.Logger
}

type UpdateTelegramPostJob struct {
	env UpdateTelegramPostEnv
}

func New(env UpdateTelegramPostEnv) *UpdateTelegramPostJob {
	task := UpdateTelegramPostJob{env: env}

	jobTimeout := time.Minute

	env.RegisterTelegramJob(ID, func(ctx context.Context, m []byte) error {
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
					"job", ID,
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

	return &task
}

func (t UpdateTelegramPostJob) QueueUpdateTelegramPost(ctx context.Context, params model.TelegramUpdatePostParams) error {
	return t.env.EnqueueTelegramJob(ctx, ID, params, 0)
}
