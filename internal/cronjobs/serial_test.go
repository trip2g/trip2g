package cronjobs

import (
	"context"
	"sync"
	"testing"
	"time"

	"trip2g/internal/db"
)

// timingJob records entry/exit timestamps for each Execute call.
type timingJob struct {
	name string
	mu   sync.Mutex
	// windows holds (entry, exit) pairs in execution order
	windows [][2]time.Time
}

func (j *timingJob) Name() string                    { return j.name }
func (j *timingJob) Schedule() string                { return "0 0 * * * *" }
func (j *timingJob) ExecuteAfterStart() bool         { return false }
func (j *timingJob) Execute(_ context.Context, _ interface{}) (interface{}, error) {
	entry := time.Now()
	time.Sleep(50 * time.Millisecond)
	exit := time.Now()
	j.mu.Lock()
	j.windows = append(j.windows, [2]time.Time{entry, exit})
	j.mu.Unlock()
	return nil, nil
}

// TestSerialExecution verifies that only one cron job runs at a time.
// The two Execute windows must NOT overlap.
//
// RED: fails without execMu because both goroutines run Execute concurrently.
// GREEN: passes after execMu serialises them.
func TestSerialExecution(t *testing.T) {
	j1 := &timingJob{name: "job-a"}
	j2 := &timingJob{name: "job-b"}

	env := &mockEnv{}
	cj := &CronJobs{
		ctx:         context.Background(),
		env:         env,
		jobs:        make(map[int64]*jobItem),
		runningJobs: make(map[int64]db.CronJobExecution),
		log:         env.Logger(),
	}
	cj.jobs[1] = &jobItem{job: j1, config: db.CronJob{ID: 1}}
	cj.jobs[2] = &jobItem{job: j2, config: db.CronJob{ID: 2}}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); cj.executeJob(1) }()
	go func() { defer wg.Done(); cj.executeJob(2) }()
	wg.Wait()

	j1.mu.Lock()
	w1 := j1.windows
	j1.mu.Unlock()
	j2.mu.Lock()
	w2 := j2.windows
	j2.mu.Unlock()

	if len(w1) == 0 || len(w2) == 0 {
		t.Fatal("at least one job did not execute")
	}

	a := w1[0]
	b := w2[0]

	// Windows must not overlap: one must finish before the other starts.
	overlap := a[0].Before(b[1]) && b[0].Before(a[1])
	if overlap {
		t.Errorf("jobs ran concurrently: job-a [%v, %v] overlaps job-b [%v, %v]",
			a[0].Format(time.RFC3339Nano), a[1].Format(time.RFC3339Nano),
			b[0].Format(time.RFC3339Nano), b[1].Format(time.RFC3339Nano))
	}
}
