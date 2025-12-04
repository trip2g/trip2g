package refreshtelegramappconfig

import (
	"context"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/logger"
)

type Env interface {
	Logger() logger.Logger
	ListAllTelegramAccounts(ctx context.Context) ([]db.TelegramAccount, error)
	TelegramAccountGetAppConfig(ctx context.Context, accountID int64) (string, error)
	UpdateTelegramAccountAppConfig(ctx context.Context, arg db.UpdateTelegramAccountAppConfigParams) error
}

type Result struct {
	UpdatedCount int
	Errors       []error
}

func Resolve(ctx context.Context, env Env) (*Result, error) {
	log := logger.WithPrefix(env.Logger(), "refreshtelegramappconfig:")

	accounts, err := env.ListAllTelegramAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list telegram accounts: %w", err)
	}

	result := &Result{}

	for _, account := range accounts {
		if account.Enabled == 0 {
			continue
		}

		appConfig, configErr := env.TelegramAccountGetAppConfig(ctx, account.ID)
		if configErr != nil {
			log.Error("failed to get app config",
				"account_id", account.ID,
				"phone", account.Phone,
				"error", configErr,
			)
			result.Errors = append(result.Errors, fmt.Errorf("account %d: %w", account.ID, configErr))
			continue
		}

		if appConfig == "" {
			continue
		}

		err = env.UpdateTelegramAccountAppConfig(ctx, db.UpdateTelegramAccountAppConfigParams{
			AppConfig: appConfig,
			ID:        account.ID,
		})
		if err != nil {
			log.Error("failed to update app config",
				"account_id", account.ID,
				"error", err,
			)
			result.Errors = append(result.Errors, fmt.Errorf("account %d update: %w", account.ID, err))
			continue
		}

		log.Info("updated app config",
			"account_id", account.ID,
			"phone", account.Phone,
		)
		result.UpdatedCount++
	}

	return result, nil
}
