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

const ID = "send_message"

type SendTelegramPostEnv interface {
	Env

	RegisterTelegramJob(id string, handler func(ctx context.Context, m []byte) error)
	EnqueueTelegramJob(ctx context.Context, jobID string, data any) error
	Logger() logger.Logger
}

type SendTelegramPostJob struct {
	env SendTelegramPostEnv
}

func New(env SendTelegramPostEnv) *SendTelegramPostJob {
	task := SendTelegramPostJob{env: env}

	jobTimeout := time.Minute

	env.RegisterTelegramJob(ID, func(ctx context.Context, m []byte) error {
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

func (t SendTelegramPostJob) QueueSendTelegramPost(ctx context.Context, params model.TelegramSendPostParams) error {
	return t.env.EnqueueTelegramJob(ctx, ID, params)
}
