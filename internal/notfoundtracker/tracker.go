package notfoundtracker

import (
	"context"
	"fmt"
)

type Env interface {
}

type Tracker struct {
	env Env
}

func New(ctx context.Context, env Env) (*Tracker, error) {
	t := Tracker{env: env}

	err := t.Refresh(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh tracker: %w", err)
	}

	return &t, nil
}

func (t *Tracker) Refresh(ctx context.Context) error {
	return nil
}

func (t *Tracker) Track(path string) error {
	return nil
}
