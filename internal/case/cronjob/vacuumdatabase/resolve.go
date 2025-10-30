package vacuumdatabase

import (
	"context"
	"fmt"
	"time"
)

type Env interface {
	VacuumDB(ctx context.Context) error
	Now() time.Time
}

type Filter struct {
	// No filters needed for this job
}

type Result struct {
	Success  bool
	Duration time.Duration
}

func Resolve(ctx context.Context, env Env, filter Filter) (*Result, error) {
	startTime := env.Now()

	err := env.VacuumDB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to vacuum database: %w", err)
	}

	duration := env.Now().Sub(startTime)

	return &Result{
		Success:  true,
		Duration: duration,
	}, nil
}
