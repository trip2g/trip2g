package refreshtelegramaccounts

import (
	"context"
	"fmt"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/tgtd"
)

type Env interface {
	Logger() logger.Logger
	ListAllTelegramAccounts(ctx context.Context) ([]db.TelegramAccount, error)
	TelegramAccountGetAppConfig(ctx context.Context, accountID int64) (string, error)
	TelegramAccountGetUserInfo(ctx context.Context, accountID int64) (*tgtd.UserInfo, error)
	UpdateTelegramAccountAppConfig(ctx context.Context, arg db.UpdateTelegramAccountAppConfigParams) error
	UpdateTelegramAccount(ctx context.Context, arg db.UpdateTelegramAccountParams) error
}

type Result struct {
	UpdatedCount int
	Errors       []error
}

func Resolve(ctx context.Context, env Env) (*Result, error) {
	log := logger.WithPrefix(env.Logger(), "refreshtelegramaccounts:")

	accounts, err := env.ListAllTelegramAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list telegram accounts: %w", err)
	}

	result := &Result{}

	for _, account := range accounts {
		if account.Enabled == 0 {
			continue
		}

		updated := false

		// Update app config
		appConfig, configErr := env.TelegramAccountGetAppConfig(ctx, account.ID)
		if configErr != nil {
			log.Error("failed to get app config",
				"account_id", account.ID,
				"phone", account.Phone,
				"error", configErr,
			)
			result.Errors = append(result.Errors, fmt.Errorf("account %d config: %w", account.ID, configErr))
		} else if appConfig != "" {
			err = env.UpdateTelegramAccountAppConfig(ctx, db.UpdateTelegramAccountAppConfigParams{
				AppConfig: appConfig,
				ID:        account.ID,
			})
			if err != nil {
				log.Error("failed to update app config",
					"account_id", account.ID,
					"error", err,
				)
				result.Errors = append(result.Errors, fmt.Errorf("account %d config update: %w", account.ID, err))
			} else {
				updated = true
			}
		}

		// Update premium status
		userInfo, userErr := env.TelegramAccountGetUserInfo(ctx, account.ID)
		//nolint:nestif // complex error handling with multiple update paths
		if userErr != nil {
			log.Error("failed to get user info",
				"account_id", account.ID,
				"phone", account.Phone,
				"error", userErr,
			)
			result.Errors = append(result.Errors, fmt.Errorf("account %d user info: %w", account.ID, userErr))
		} else {
			newPremium := int64(0)
			if userInfo.IsPremium {
				newPremium = 1
			}

			if account.IsPremium != newPremium {
				err = env.UpdateTelegramAccount(ctx, db.UpdateTelegramAccountParams{
					ID:        account.ID,
					IsPremium: &newPremium,
				})
				if err != nil {
					log.Error("failed to update premium status",
						"account_id", account.ID,
						"error", err,
					)
					result.Errors = append(result.Errors, fmt.Errorf("account %d premium update: %w", account.ID, err))
				} else {
					log.Info("updated premium status",
						"account_id", account.ID,
						"phone", account.Phone,
						"is_premium", userInfo.IsPremium,
					)
					updated = true
				}
			}
		}

		if updated {
			log.Info("refreshed account",
				"account_id", account.ID,
				"phone", account.Phone,
			)
			result.UpdatedCount++
		}
	}

	return result, nil
}
