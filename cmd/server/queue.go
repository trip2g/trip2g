package main

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	"maragu.dev/goqite"
	"maragu.dev/goqite/jobs"
)

type appQueue struct {
	name string

	q *goqite.Queue

	rootCtx context.Context

	cancel context.CancelFunc
	runner *jobs.Runner

	mu sync.Mutex
}

func (a *app) createQueue(ctx context.Context, name string, runnerOpts jobs.NewRunnerOpts) *appQueue {
	q := goqite.New(goqite.NewOpts{
		DB:   a.writeConn,
		Name: name,
	})

	runnerOpts.Queue = q
	runnerOpts.Log = logger.WithPrefix(a.log, fmt.Sprintf("%s_runner:", name))

	runner := jobs.NewRunner(runnerOpts)

	appQ := appQueue{
		q:       q,
		rootCtx: ctx, // for app graceful shutdown
		name:    name,
		runner:  runner,
	}

	if a.appQueues == nil {
		a.appQueues = make(map[string]*appQueue)
	}

	a.appQueues[name] = &appQ

	return &appQ
}

func (a *appQueue) stop() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.cancel()
	a.cancel = nil
}

func (a *appQueue) start() {
	a.mu.Lock()
	defer a.mu.Unlock()

	ctx, cancel := context.WithCancel(a.rootCtx)
	a.cancel = cancel

	go a.runner.Start(ctx)
}

func (a *appQueue) isStopped() bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.cancel == nil
}

func (a *appQueue) toModel() *model.BackgroundQueue {
	return &model.BackgroundQueue{
		Name:    a.name,
		Stopped: a.isStopped(),
	}
}

func (a *app) getBackgroundQueue(name string) (*appQueue, error) {
	q, ok := a.appQueues[name]
	if !ok {
		return nil, fmt.Errorf("queue %s not found", name)
	}

	return q, nil
}

func (a *app) GetBackgroundQueue(ctx context.Context, name string) (*model.BackgroundQueue, error) {
	q, err := a.getBackgroundQueue(name)
	if err != nil {
		return nil, err
	}

	return q.toModel(), nil
}

func (a *app) ListBackgroundQueues(ctx context.Context) []model.BackgroundQueue {
	// Get and sort queue names
	names := make([]string, 0, len(a.appQueues))
	for name := range a.appQueues {
		names = append(names, name)
	}
	sort.Strings(names)

	// Build result in sorted order
	queues := make([]model.BackgroundQueue, 0, len(a.appQueues))
	for _, name := range names {
		queues = append(queues, *a.appQueues[name].toModel())
	}

	return queues
}

func (a *app) StopBackgroundQueue(ctx context.Context, name string) error {
	q, err := a.getBackgroundQueue(name)
	if err != nil {
		return err
	}

	q.stop()
	return nil
}

func (a *app) StartBackgroundQueue(ctx context.Context, name string) error {
	q, err := a.getBackgroundQueue(name)
	if err != nil {
		return err
	}

	q.start()
	return nil
}

func (a *app) ClearBackgroundQueue(ctx context.Context, name string) (int64, error) {
	q, err := a.getBackgroundQueue(name)
	if err != nil {
		return 0, err
	}

	// Remember if queue was running
	wasRunning := !q.isStopped()

	// Stop queue if running
	if wasRunning {
		q.stop()
	}

	// Delete all jobs from this queue
	result, err := a.writeConn.ExecContext(ctx, "DELETE FROM goqite WHERE queue = ?", name)
	if err != nil {
		// Try to restart queue if it was running
		if wasRunning {
			q.start()
		}
		return 0, fmt.Errorf("failed to delete jobs: %w", err)
	}

	deletedCount, err := result.RowsAffected()
	if err != nil {
		// Try to restart queue if it was running
		if wasRunning {
			q.start()
		}
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	// Restart queue if it was running
	if wasRunning {
		q.start()
	}

	return deletedCount, nil
}
