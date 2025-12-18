package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
	"trip2g/internal/appreq"
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

	logger logger.Logger

	mu sync.Mutex
}

// QueueOpts combines options for queue creation.
type QueueOpts struct {
	Limit        int           // Max concurrent jobs (default: 1)
	PollInterval time.Duration // How often to poll for new jobs
	Extend       time.Duration // How often to extend job timeout
	MaxReceive   int           // Max receive count before message is dropped (default: 3)
}

func (a *app) createQueue(ctx context.Context, name string, opts QueueOpts) *appQueue {
	queueOpts := goqite.NewOpts{
		DB:   a.writeConn,
		Name: name,
	}
	if opts.MaxReceive > 0 {
		queueOpts.MaxReceive = opts.MaxReceive
	}
	q := goqite.New(queueOpts)

	logger := logger.WithPrefix(a.log, name+":")

	if opts.PollInterval < 50*time.Millisecond {
		panic("too small poll interval. Are you sure?")
	}

	runnerOpts := jobs.NewRunnerOpts{
		Limit:        opts.Limit,
		PollInterval: opts.PollInterval,
		Extend:       opts.Extend,
		Queue:        q,
		Log:          logger,
	}

	runner := jobs.NewRunner(runnerOpts)

	appQ := appQueue{
		q:       q,
		rootCtx: ctx, // for app graceful shutdown
		name:    name,
		runner:  runner,
		logger:  logger,
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
	// Handle special case: stop all queues
	if name == "*" {
		for _, q := range a.appQueues {
			q.stop()
		}
		return nil
	}

	q, err := a.getBackgroundQueue(name)
	if err != nil {
		return err
	}

	q.stop()
	return nil
}

func (a *app) StartBackgroundQueue(ctx context.Context, name string) error {
	// Handle special case: start all queues
	if name == "*" {
		for _, q := range a.appQueues {
			q.start()
		}
		return nil
	}

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

func (a *app) enqueueJobToQ(ctx context.Context, aq *appQueue, job model.BackgroundTask) error {
	rawData, err := json.Marshal(job.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal job data: %w", err)
	}

	gqMsg := goqite.Message{
		Body:     rawData,
		Priority: job.Priority,
	}

	// First check context for transactional env (works for both HTTP and background jobs)
	if txEnv, ok := ctx.Value(txEnvKey).(*app); ok && txEnv.currentTx != nil {
		a.log.Debug("enqueueing in context tx env", "job_id", job.ID, "data", string(rawData), "queue", aq.name)

		_, err = jobs.CreateTx(ctx, txEnv.currentTx, aq.q, job.ID, gqMsg)
		return err
	}

	// Fallback: check request context (for HTTP requests)
	req, err := appreq.FromCtx(ctx)
	if err != nil && !errors.Is(err, appreq.ErrNotFound) {
		return fmt.Errorf("failed to get request from context: %w", err)
	}

	if req != nil {
		env, ok := req.Env.(*app)
		if ok && env.CurrentTx() != nil {
			a.log.Debug("enqueueing in request env", "job_id", job.ID, "data", string(rawData), "queue", aq.name)

			_, err = jobs.CreateTx(ctx, env.currentTx, aq.q, job.ID, gqMsg)
			return err
		}
	}

	// Fallback: check global app (should rarely be used now)
	if a.currentTx != nil {
		a.log.Debug("enqueueing in app.currentTx", "job_id", job.ID, "data", string(rawData), "queue", aq.name)

		_, err = jobs.CreateTx(ctx, a.currentTx, aq.q, job.ID, gqMsg)
		return err
	}

	a.log.Debug("enqueueing in global env", "job_id", job.ID, "data", string(rawData), "queue", aq.name)

	_, err = jobs.Create(ctx, aq.q, job.ID, gqMsg)
	return err
}

func (a *app) EnqueueJob(ctx context.Context, job model.BackgroundTask) error {
	switch job.Queue {
	case model.BackgroundDefaultQueue:
		return a.enqueueJobToQ(ctx, a.globalQueue, job)
	case model.BackgroundTelegramJobQueue:
		return a.enqueueJobToQ(ctx, a.telegramTaskQueue, job)
	case model.BackgroundTelegramBotAPIQueue:
		return a.enqueueJobToQ(ctx, a.telegramBotAPIQueue, job)
	case model.BackgroundTelegramAccountAPIQueue:
		return a.enqueueJobToQ(ctx, a.telegramAccountAPIQueue, job)
	case model.BackgroundTelegramLongRunningQueue:
		return a.enqueueJobToQ(ctx, a.telegramLongRunningQueue, job)
	}

	return fmt.Errorf("unknown queue: %d", job.Queue)
}

func (a *app) RegisterJob(qID model.BackgroundQueueID, id string, handler func(ctx context.Context, m []byte) error) {
	a.log.Info("registering job handler", "id", id, "queue", qID.String())

	switch qID {
	case model.BackgroundDefaultQueue:
		a.globalQueue.runner.Register(id, handler)
	case model.BackgroundTelegramJobQueue:
		a.telegramTaskQueue.runner.Register(id, handler)
	case model.BackgroundTelegramBotAPIQueue:
		a.telegramBotAPIQueue.runner.Register(id, handler)
	case model.BackgroundTelegramAccountAPIQueue:
		a.telegramAccountAPIQueue.runner.Register(id, handler)
	case model.BackgroundTelegramLongRunningQueue:
		a.telegramLongRunningQueue.runner.Register(id, handler)
	default:
		panic(fmt.Sprintf("unknown queue: %d", qID))
	}
}
