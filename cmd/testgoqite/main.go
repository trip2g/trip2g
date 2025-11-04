package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/zerologger"

	"maragu.dev/goqite"
	"maragu.dev/goqite/jobs"
)

func main() {
	ctx := context.Background()
	log := zerologger.New("debug", true)

	// Create database in /tmp
	dbFile := "/tmp/test_goqite.db"
	log.Info("creating test database", "file", dbFile)

	// Create read connection
	readConn, err := db.Setup(db.SetupConfig{
		DatabaseFile: dbFile,
		Logger:       log,
		ReadOnly:     true,
	})
	if err != nil {
		panic(fmt.Errorf("failed to setup read database: %w", err))
	}
	defer readConn.Close()

	// Create write connection
	writeConn, err := db.Setup(db.SetupConfig{
		DatabaseFile: dbFile,
		Logger:       log,
		SkipDump:     true,
	})
	if err != nil {
		panic(fmt.Errorf("failed to setup write database: %w", err))
	}
	defer writeConn.Close()

	// readQueries := db.New(readConn)
	// writeQueries := db.NewWriteQueries(writeConn)

	log.Info("database connections created")

	// Create two separate queues
	queue1 := goqite.New(goqite.NewOpts{
		DB:   writeConn,
		Name: "queue1",
	})

	queue2 := goqite.New(goqite.NewOpts{
		DB:   writeConn,
		Name: "queue2",
	})

	log.Info("created two goqite queues")

	// Create two separate runners with production-like settings to reproduce SQLITE_BUSY
	runner1 := jobs.NewRunner(jobs.NewRunnerOpts{
		Limit:        1, // Like tg_jobs in production
		Log:          logger.WithPrefix(log, "runner1:"),
		PollInterval: time.Millisecond * 901, // Like global_jobs in production
		Queue:        queue1,
	})

	runner2 := jobs.NewRunner(jobs.NewRunnerOpts{
		Limit:        1, // Like tg_jobs in production
		Log:          logger.WithPrefix(log, "runner2:"),
		PollInterval: time.Second * 1, // Like tg_jobs in production (1000ms)
		Queue:        queue2,
	})

	// Register job handlers that sleep less (more work)
	runner1.Register("job1", func(ctx context.Context, m []byte) error {
		log.Debug("runner1 processing job", "data", string(m))
		time.Sleep(500 * time.Millisecond) // Less sleep = more job processing
		log.Debug("runner1 finished job", "data", string(m))
		return nil
	})

	runner2.Register("job2", func(ctx context.Context, m []byte) error {
		log.Debug("runner2 processing job", "data", string(m))
		time.Sleep(500 * time.Millisecond) // Less sleep = more job processing
		log.Debug("runner2 finished job", "data", string(m))
		return nil
	})

	log.Info("registered job handlers")

	// Start both runners
	go runner1.Start(ctx)
	go runner2.Start(ctx)

	log.Info("started both runners")

	// Wait for runners to start
	time.Sleep(time.Second)

	var wg sync.WaitGroup
	var jobErrors sync.Map

	job1Count := 0
	job2Count := 0

	var job1Mu sync.Mutex
	var job2Mu sync.Mutex

	// Skip insert/list operations to focus on queue conflicts
	log.Info("skipping user insert/list operations to focus on queue conflicts")

	// Goroutine to enqueue jobs to queue1 (very frequently to increase load)
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(10 * time.Millisecond) // Much faster to create contention
		defer ticker.Stop()

		timeout := time.After(60 * time.Second) // Run for 60 seconds

		for {
			select {
			case <-timeout:
				log.Info("job1 enqueue goroutine stopping")
				return
			case <-ticker.C:
				jobData := []byte(fmt.Sprintf("job1-%d", rand.Int()))
				_, err := jobs.Create(ctx, queue1, "job1", goqite.Message{Body: jobData})
				if err != nil {
					log.Error("failed to enqueue job1", "error", err)
					jobErrors.Store(fmt.Sprintf("job1-%d", time.Now().UnixNano()), err.Error())
				} else {
					job1Mu.Lock()
					job1Count++
					job1Mu.Unlock()
				}
			}
		}
	}()

	// Goroutine to enqueue jobs to queue2 (very frequently to increase load)
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(10 * time.Millisecond) // Much faster to create contention
		defer ticker.Stop()

		timeout := time.After(60 * time.Second) // Run for 60 seconds

		for {
			select {
			case <-timeout:
				log.Info("job2 enqueue goroutine stopping")
				return
			case <-ticker.C:
				jobData := []byte(fmt.Sprintf("job2-%d", rand.Int()))
				_, err := jobs.Create(ctx, queue2, "job2", goqite.Message{Body: jobData})
				if err != nil {
					log.Error("failed to enqueue job2", "error", err)
					jobErrors.Store(fmt.Sprintf("job2-%d", time.Now().UnixNano()), err.Error())
				} else {
					job2Mu.Lock()
					job2Count++
					job2Mu.Unlock()
				}
			}
		}
	}()

	log.Info("all goroutines started, waiting for 60 seconds")

	// Wait for all goroutines to finish
	wg.Wait()

	log.Info("test completed",
		"job1_enqueued", job1Count,
		"job2_enqueued", job2Count,
	)

	// Count errors
	jobErrorCount := 0
	jobErrors.Range(func(key, value interface{}) bool {
		jobErrorCount++
		return true
	})

	log.Info("error summary",
		"job_errors", jobErrorCount,
	)

	if jobErrorCount > 0 {
		log.Info("ALL job errors:")
		jobErrors.Range(func(key, value interface{}) bool {
			log.Error("job error", "key", key, "error", value)
			return true
		})
	} else {
		log.Info("✅ NO ERRORS! Test passed.")
	}

	// Give runners time to finish processing
	time.Sleep(5 * time.Second)

	log.Info("test finished")
}
