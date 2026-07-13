package cronjobs

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
)

// mockEnv implements Env with controllable hooks for DB calls.
type mockEnv struct {
	updateRunningFn func(ctx context.Context, params db.UpdateRunningCronJobExecutionsParams) error
	insertExecFn    func(arg db.InsertCronJobExecutionParams) (db.CronJobExecution, error)
	updateExecFn    func(arg db.UpdateCronJobExecutionParams) (db.CronJobExecution, error)
}

func (m *mockEnv) UpdateRunningCronJobExecutions(ctx context.Context, p db.UpdateRunningCronJobExecutionsParams) error {
	if m.updateRunningFn != nil {
		return m.updateRunningFn(ctx, p)
	}
	return nil
}

func (m *mockEnv) InsertCronJobExecution(_ context.Context, arg db.InsertCronJobExecutionParams) (db.CronJobExecution, error) {
	if m.insertExecFn != nil {
		return m.insertExecFn(arg)
	}
	return db.CronJobExecution{ID: 1, JobID: arg.JobID, Status: arg.Status}, nil
}

func (m *mockEnv) UpdateCronJobExecution(_ context.Context, arg db.UpdateCronJobExecutionParams) (db.CronJobExecution, error) {
	if m.updateExecFn != nil {
		return m.updateExecFn(arg)
	}
	return db.CronJobExecution{ID: arg.ID, Status: arg.Status}, nil
}

func (m *mockEnv) UpdateCronJobLastExec(_ context.Context, _ int64) error          { return nil }
func (m *mockEnv) UpsertCronJob(_ context.Context, _ db.UpsertCronJobParams) error { return nil }
func (m *mockEnv) CronJobByName(_ context.Context, _ string) (db.CronJob, error) {
	return db.CronJob{}, nil
}
func (m *mockEnv) ListAllCronJobs(_ context.Context) ([]db.CronJob, error)    { return nil, nil }
func (m *mockEnv) DeleteCronJobByName(_ context.Context, _ string) error      { return nil }
func (m *mockEnv) Logger() logger.Logger                                      { return &logger.DummyLogger{} }
func (m *mockEnv) CronScheduleOverride(_ string) string                       { return "" }
func (m *mockEnv) EnqueueJob(_ context.Context, _ model.BackgroundTask) error { return nil }
func (m *mockEnv) RegisterJob(_ model.BackgroundQueueID, _ string, _ func(context.Context, []byte) error) {
}

type noopJob struct{ name string }

func (j *noopJob) Name() string                                                  { return j.name }
func (j *noopJob) Schedule() string                                              { return "0 0 * * * *" }
func (j *noopJob) ExecuteAfterStart() bool                                       { return false }
func (j *noopJob) Execute(_ context.Context) (interface{}, error)                { return nil, nil }

// newTestCronJobs creates a CronJobs with pre-populated jobs map, bypassing New().
func newTestCronJobs(env Env, jobIDs []int64) *CronJobs {
	cj := &CronJobs{
		ctx:         context.Background(),
		env:         env,
		jobs:        make(map[int64]*jobItem),
		runningJobs: make(map[int64]db.CronJobExecution),
		log:         env.Logger(),
	}
	for _, id := range jobIDs {
		cj.jobs[id] = &jobItem{
			job:    &noopJob{name: "test"},
			config: db.CronJob{ID: id},
		}
	}
	return cj
}

// TestExecuteJob_DoesNotHoldMutexDuringDBOps verifies that runningMU is released
// before DB calls so a concurrent executeJob for a different job is not blocked.
//
// With the bug: job1 holds runningMU during UpdateRunningCronJobExecutions;
// job2 blocks on runningMU.Lock() and cannot signal job2Reached within the timeout.
// After the fix: job2 acquires the mutex independently and signals on time.
func TestExecuteJob_DoesNotHoldMutexDuringDBOps(t *testing.T) {
	// job1 blocks inside UpdateRunningCronJobExecutions until we release it.
	// job2 signals immediately when it enters UpdateRunningCronJobExecutions.
	job1Block := make(chan struct{})
	job2Reached := make(chan struct{})

	env := &mockEnv{
		updateRunningFn: func(_ context.Context, p db.UpdateRunningCronJobExecutionsParams) error {
			switch p.JobID {
			case 1:
				<-job1Block // simulate slow SQLite write lock on job1
			case 2:
				close(job2Reached) // job2 reached DB call — mutex was released
			}
			return nil
		},
	}
	cj := newTestCronJobs(env, []int64{1, 2})

	// Start job1 — it will block inside UpdateRunningCronJobExecutions.
	go cj.executeJob(1)

	// Give job1 time to acquire the mutex and enter the DB call.
	time.Sleep(10 * time.Millisecond)

	// Start job2 concurrently. If runningMU is still held by job1, job2 cannot
	// call UpdateRunningCronJobExecutions and job2Reached will not be closed.
	go cj.executeJob(2)

	select {
	case <-job2Reached:
		// job2 reached DB call while job1 is still blocked — fix is working
	case <-time.After(100 * time.Millisecond):
		t.Fatal("job2 blocked: executeJob holds runningMU during DB operations")
	}

	close(job1Block) // unblock job1 to let goroutines exit cleanly
}

// gateJob counts Execute calls atomically (safe for concurrent executeJob tests).
type gateJob struct {
	name      string
	execCount atomic.Int32
}

func (j *gateJob) Name() string            { return j.name }
func (j *gateJob) Schedule() string        { return "0 0 * * * *" }
func (j *gateJob) ExecuteAfterStart() bool { return false }
func (j *gateJob) Execute(_ context.Context) (interface{}, error) {
	j.execCount.Add(1)
	return nil, nil
}

// TestExecuteJob_ConcurrentSameJobExecutesOnce verifies the same-job dedup is
// atomic: two concurrent executeJob calls for the same jobID must run Execute
// exactly once. The runningJobs check and the running-slot insert are separated
// by DB writes, so both callers can pass the check before either marks the job
// as running.
func TestExecuteJob_ConcurrentSameJobExecutesOnce(t *testing.T) {
	insertEntered := make(chan struct{}, 2)
	insertRelease := make(chan struct{})

	var execID atomic.Int64
	env := &mockEnv{
		insertExecFn: func(arg db.InsertCronJobExecutionParams) (db.CronJobExecution, error) {
			insertEntered <- struct{}{}
			<-insertRelease
			return db.CronJobExecution{ID: execID.Add(1), JobID: arg.JobID, Status: arg.Status}, nil
		},
	}

	job := &gateJob{name: "dup"}
	cj := newTestCronJobs(env, nil)
	cj.jobs[1] = &jobItem{job: job, config: db.CronJob{ID: 1, Name: "dup", Enabled: true}}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); cj.executeJob(1) }()

	// Wait until the first call is past the runningJobs check, inside the insert.
	<-insertEntered

	go func() { defer wg.Done(); cj.executeJob(1) }()

	// With correct dedup the second call returns early and never reaches the
	// insert; give it a moment either way, then release the blocked insert(s).
	select {
	case <-insertEntered:
	case <-time.After(100 * time.Millisecond):
	}
	close(insertRelease)
	wg.Wait()

	if got := job.execCount.Load(); got != 1 {
		t.Errorf("Execute ran %d time(s) for concurrent same-job calls, expected 1", got)
	}
}

// TestExecuteJob_CtxCancelledAfterLock_MarksExecutionTerminal verifies that
// when shutdown cancels cj.ctx after the execution row was inserted, the row
// is marked failed instead of being left RUNNING forever (the in-memory
// reservation is cleaned by the defer, but the DB row was not).
func TestExecuteJob_CtxCancelledAfterLock_MarksExecutionTerminal(t *testing.T) {
	var updates []db.UpdateCronJobExecutionParams
	env := &mockEnv{
		updateExecFn: func(arg db.UpdateCronJobExecutionParams) (db.CronJobExecution, error) {
			updates = append(updates, arg)
			return db.CronJobExecution{ID: arg.ID, Status: arg.Status}, nil
		},
	}
	cj := newTestCronJobs(env, []int64{1})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cj.ctx = ctx

	_, err := cj.executeJob(1)
	if err == nil {
		t.Fatal("expected error from executeJob with cancelled context")
	}

	if len(updates) != 1 {
		t.Fatalf("expected 1 UpdateCronJobExecution call marking the row terminal, got %d", len(updates))
	}
	if updates[0].ID != 1 {
		t.Errorf("expected update for execution ID 1, got %d", updates[0].ID)
	}
	if updates[0].Status != JobStatusFailed {
		t.Errorf("expected terminal status %d (failed), got %d", JobStatusFailed, updates[0].Status)
	}
	if updates[0].ErrorMessage == nil || *updates[0].ErrorMessage == "" {
		t.Error("expected a non-empty error message on the cancelled execution row")
	}
}

// TestExecuteJob_DuplicateCallerGetsAlreadyRunning: the loser of the same-job
// dedup race must not receive the in-memory placeholder {ID:0, Status:RUNNING}
// as if it were a real execution row — it must get ErrJobAlreadyRunning.
func TestExecuteJob_DuplicateCallerGetsAlreadyRunning(t *testing.T) {
	insertEntered := make(chan struct{}, 2)
	insertRelease := make(chan struct{})

	var execID atomic.Int64
	env := &mockEnv{
		insertExecFn: func(arg db.InsertCronJobExecutionParams) (db.CronJobExecution, error) {
			insertEntered <- struct{}{}
			<-insertRelease
			return db.CronJobExecution{ID: execID.Add(1), JobID: arg.JobID, Status: arg.Status}, nil
		},
	}

	job := &gateJob{name: "dup"}
	cj := newTestCronJobs(env, nil)
	cj.jobs[1] = &jobItem{job: job, config: db.CronJob{ID: 1, Name: "dup", Enabled: true}}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); cj.executeJob(1) }()

	// Wait until the first call is past the dedup check, blocked in the insert —
	// runningJobs now holds the {ID:0} placeholder.
	<-insertEntered

	exec, err := cj.executeJob(1)
	close(insertRelease)
	wg.Wait()

	if !errors.Is(err, ErrJobAlreadyRunning) {
		t.Errorf("duplicate caller: expected ErrJobAlreadyRunning, got exec=%+v err=%v", exec, err)
	}
	if exec != nil && exec.ID == 0 {
		t.Errorf("duplicate caller must never receive the ID-0 placeholder, got %+v", exec)
	}
}

// countingJob records how many times Execute was called.
type countingJob struct {
	name       string
	afterStart bool
	execCount  int
}

func (j *countingJob) Name() string            { return j.name }
func (j *countingJob) Schedule() string        { return "0 0 * * * *" }
func (j *countingJob) ExecuteAfterStart() bool { return j.afterStart }
func (j *countingJob) Execute(_ context.Context) (interface{}, error) {
	j.execCount++
	return nil, nil
}

func TestRunStartupJobs_RunsOnlyExecuteAfterStartJobsOnce(t *testing.T) {
	jobA := &countingJob{name: "a", afterStart: true}
	jobB := &countingJob{name: "b", afterStart: false}
	jobC := &countingJob{name: "c", afterStart: true}
	jobD := &countingJob{name: "d", afterStart: true} // disabled in DB config

	env := &mockEnv{}
	cj := newTestCronJobs(env, nil)
	cj.jobs[1] = &jobItem{job: jobA, config: db.CronJob{ID: 1, Name: "a", Enabled: true}}
	cj.jobs[2] = &jobItem{job: jobB, config: db.CronJob{ID: 2, Name: "b", Enabled: true}}
	cj.jobs[3] = &jobItem{job: jobC, config: db.CronJob{ID: 3, Name: "c", Enabled: true}}
	cj.jobs[4] = &jobItem{job: jobD, config: db.CronJob{ID: 4, Name: "d", Enabled: false}}

	cj.RunStartupJobs(context.Background(), true)

	if jobA.execCount != 1 {
		t.Errorf("job a (afterStart=true): expected 1 execution, got %d", jobA.execCount)
	}
	if jobB.execCount != 0 {
		t.Errorf("job b (afterStart=false): expected 0 executions, got %d", jobB.execCount)
	}
	if jobC.execCount != 1 {
		t.Errorf("job c (afterStart=true): expected 1 execution, got %d", jobC.execCount)
	}
	if jobD.execCount != 0 {
		t.Errorf("job d (disabled): expected 0 executions, got %d", jobD.execCount)
	}
}

func TestRunStartupJobs_StopsOnCancelledContext(t *testing.T) {
	jobA := &countingJob{name: "a", afterStart: true}

	env := &mockEnv{}
	cj := newTestCronJobs(env, nil)
	cj.jobs[1] = &jobItem{job: jobA, config: db.CronJob{ID: 1, Name: "a", Enabled: true}}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cj.RunStartupJobs(ctx, true)

	if jobA.execCount != 0 {
		t.Errorf("expected 0 executions with cancelled context, got %d", jobA.execCount)
	}
}

func TestRunStartupJobs_DisabledRunsNothing(t *testing.T) {
	jobA := &countingJob{name: "a", afterStart: true}

	env := &mockEnv{}
	cj := newTestCronJobs(env, nil)
	cj.jobs[1] = &jobItem{job: jobA, config: db.CronJob{ID: 1, Name: "a", Enabled: true}}

	cj.RunStartupJobs(context.Background(), false)

	if jobA.execCount != 0 {
		t.Errorf("expected 0 executions with flag disabled, got %d", jobA.execCount)
	}
}
