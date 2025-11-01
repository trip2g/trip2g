package main

import (
	"context"
	"fmt"
	"sync"
	"trip2g/internal/logger"

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
}

func (a *appQueue) start() {
	a.mu.Lock()
	defer a.mu.Unlock()

	ctx, cancel := context.WithCancel(a.rootCtx)
	a.cancel = cancel

	go a.runner.Start(ctx)
}
