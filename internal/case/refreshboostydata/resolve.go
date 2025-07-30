package refreshboostydata

import (
	"context"
	"fmt"
	"trip2g/internal/boosty"
)

type Env interface {
	BoostyClientByCredentialsID(ctx context.Context, credentialsID int64) (boosty.Client, error)
}

func Resolve(ctx context.Context, env Env, credentialsID int64) error {
	client, err := env.BoostyClientByCredentialsID(ctx, credentialsID)
	if err != nil {
		return fmt.Errorf("failed to get Boosty client: %w", err)
	}

	subscribers, err := client.Subscribers()
	if err != nil {
		return fmt.Errorf("failed to get subscribers: %w", err)
	}

	fmt.Println("Subscribers:", subscribers)

	return nil
}
