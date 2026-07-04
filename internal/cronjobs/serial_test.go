package cronjobs

import (
	"context"
	"sync"
	"testing"
	"time"

	"trip2g/internal/db"
)

// sleepJob records entry/exit timestamps and sleeps for a fixed duration.
type sleepJob struct {
	name    string
	sleep   time.Duration
	mu      sync.Mutex
	entries []time.Time
	exits   []time.Time
}

func (j *sleepJob) Name() string           { return j.name }
func (j *sleepJob) Schedule() string        { return "0 0 * * * *" }
func (j *sleepJob) ExecuteAfterStart() bool { return false }
func (j *sleepJob) Execute(_ context.Context, _ interface{}) (interface{}, error) {
	entry := time.Now()
	j.mu.Lock()
	j.entries = append(j.entries, entry)
	j.mu.Unlock()

	time.Sleep(j.sleep)

	exit := time.Now()
	j.mu.Lock()
	j.exits = append(j.exits, exit)
	j.mu.Unlock()

	return nil, nil
}

// TestExecuteJob_SerialExecution asserts that two concurrent executeJob calls do NOT
// overlap in their Execute windows. RED without execMu, GREEN after the fix.
func TestExecuteJob_SerialExecution(t *testing.T) {
	const sleepDur = 50 * time.Millisecond

	jobA := &sleepJob{name: "jobA", sleep: sleepDur}
	jobB := &sleepJob{name: "jobB", sleep: sleepDur}

	env := &mockEnv{}
	cj := &CronJobs{
		ctx:         context.Background(),
		env:         env,
		jobs:        make(map[int64]*jobItem),
		runningJobs: make(map[int64]db.CronJobExecution),
		log:         env.Logger(),
	}
	cj.jobs[1] = &jobItem{job: jobA, config: db.CronJob{ID: 1}}
	cj.jobs[2] = &jobItem{job: jobB, config: db.CronJob{ID: 2}}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); cj.executeJob(1) }() //nolint:errcheck
	go func() { defer wg.Done(); cj.executeJob(2) }() //nolint:errcheck
	wg.Wait()

	// Each job must have run exactly once.
	if len(jobA.entries) != 1 || len(jobA.exits) != 1 {
		t.Fatalf("jobA ran %d time(s), expected 1", len(jobA.entries))
	}
	if len(jobB.entries) != 1 || len(jobB.exits) != 1 {
		t.Fatalf("jobB ran %d time(s), expected 1", len(jobB.entries))
	}

	aEntry, aExit := jobA.entries[0], jobA.exits[0]
	bEntry, bExit := jobB.entries[0], jobB.exits[0]

	// Windows must NOT overlap: either A finished before B started, or vice versa.
	aBeforeB := !aExit.After(bEntry)
	bBeforeA := !bExit.After(aEntry)

	if !aBeforeB && !bBeforeA {
		t.Errorf("execution windows overlapped: A[%v..%v] B[%v..%v]",
			aEntry.Format(time.RFC3339Nano),
			aExit.Format(time.RFC3339Nano),
			bEntry.Format(time.RFC3339Nano),
			bExit.Format(time.RFC3339Nano),
		)
	}
}
