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

	readQueries := db.New(readConn)
	writeQueries := db.NewWriteQueries(writeConn)

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

	// Create two separate runners with more aggressive settings
	runner1 := jobs.NewRunner(jobs.NewRunnerOpts{
		Limit:        10, // More concurrent jobs
		Log:          logger.WithPrefix(log, "runner1:"),
		PollInterval: time.Millisecond * 10, // Poll more frequently
		Queue:        queue1,
	})

	runner2 := jobs.NewRunner(jobs.NewRunnerOpts{
		Limit:        10, // More concurrent jobs
		Log:          logger.WithPrefix(log, "runner2:"),
		PollInterval: time.Millisecond * 10, // Poll more frequently
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
	var insertErrors sync.Map
	var listErrors sync.Map
	var jobErrors sync.Map

	insertCount := 0
	listCount := 0
	job1Count := 0
	job2Count := 0

	var insertMu sync.Mutex
	var listMu sync.Mutex
	var job1Mu sync.Mutex
	var job2Mu sync.Mutex

	// Goroutine to insert users every 2ms (more aggressive)
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(2 * time.Millisecond)
		defer ticker.Stop()

		timeout := time.After(30 * time.Second)

		for {
			select {
			case <-timeout:
				log.Info("insert goroutine stopping")
				return
			case <-ticker.C:
				email := fmt.Sprintf("user%d@example.com", time.Now().UnixNano())
				_, err := writeQueries.InsertUserWithEmail(ctx, db.InsertUserWithEmailParams{
					Email:      email,
					CreatedVia: "test",
				})
				if err != nil {
					log.Error("failed to insert user", "error", err, "email", email)
					insertErrors.Store(time.Now().UnixNano(), err.Error())
				} else {
					insertMu.Lock()
					insertCount++
					insertMu.Unlock()
				}
			}
		}
	}()

	// Goroutine to list users every 1ms (more aggressive)
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(1 * time.Millisecond)
		defer ticker.Stop()

		timeout := time.After(30 * time.Second)

		for {
			select {
			case <-timeout:
				log.Info("list goroutine stopping")
				return
			case <-ticker.C:
				users, err := readQueries.ListAllUsers(ctx)
				if err != nil {
					log.Error("failed to list users", "error", err)
					listErrors.Store(time.Now().UnixNano(), err.Error())
				} else {
					listMu.Lock()
					listCount++
					listMu.Unlock()
					log.Debug("listed users", "count", len(users))
				}
			}
		}
	}()

	// Goroutine to enqueue jobs to queue1 (more frequently)
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		timeout := time.After(30 * time.Second)

		for {
			select {
			case <-timeout:
				log.Info("job1 enqueue goroutine stopping")
				return
			case <-ticker.C:
				jobData := []byte(fmt.Sprintf("job1-%d", rand.Int()))
				err := jobs.Create(ctx, queue1, "job1", jobData)
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

	// Goroutine to enqueue jobs to queue2 (more frequently)
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		timeout := time.After(30 * time.Second)

		for {
			select {
			case <-timeout:
				log.Info("job2 enqueue goroutine stopping")
				return
			case <-ticker.C:
				jobData := []byte(fmt.Sprintf("job2-%d", rand.Int()))
				err := jobs.Create(ctx, queue2, "job2", jobData)
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

	log.Info("all goroutines started, waiting for 30 seconds")

	// Wait for all goroutines to finish
	wg.Wait()

	log.Info("test completed",
		"inserts", insertCount,
		"lists", listCount,
		"job1_enqueued", job1Count,
		"job2_enqueued", job2Count,
	)

	// Count errors
	insertErrorCount := 0
	insertErrors.Range(func(key, value interface{}) bool {
		insertErrorCount++
		return true
	})

	listErrorCount := 0
	listErrors.Range(func(key, value interface{}) bool {
		listErrorCount++
		return true
	})

	jobErrorCount := 0
	jobErrors.Range(func(key, value interface{}) bool {
		jobErrorCount++
		return true
	})

	log.Info("error summary",
		"insert_errors", insertErrorCount,
		"list_errors", listErrorCount,
		"job_errors", jobErrorCount,
	)

	if insertErrorCount > 0 {
		log.Info("sample insert errors (first 3):")
		count := 0
		insertErrors.Range(func(key, value interface{}) bool {
			log.Error("insert error", "error", value)
			count++
			return count < 3
		})
	}

	if listErrorCount > 0 {
		log.Info("sample list errors (first 3):")
		count := 0
		listErrors.Range(func(key, value interface{}) bool {
			log.Error("list error", "error", value)
			count++
			return count < 3
		})
	}

	if jobErrorCount > 0 {
		log.Info("sample job errors (first 3):")
		count := 0
		jobErrors.Range(func(key, value interface{}) bool {
			log.Error("job error", "key", key, "error", value)
			count++
			return count < 3
		})
	}

	// Give runners time to finish processing
	time.Sleep(5 * time.Second)

	log.Info("test finished")
}
