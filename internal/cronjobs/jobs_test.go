package cronjobs

import (
	"context"
	"testing"
	"time"

	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
)

// mockEnv implements Env with a controllable hook for UpdateRunningCronJobExecutions.
type mockEnv struct {
	updateRunningFn func(ctx context.Context, params db.UpdateRunningCronJobExecutionsParams) error
}

func (m *mockEnv) UpdateRunningCronJobExecutions(ctx context.Context, p db.UpdateRunningCronJobExecutionsParams) error {
	if m.updateRunningFn != nil {
		return m.updateRunningFn(ctx, p)
	}
	return nil
}

func (m *mockEnv) InsertCronJobExecution(_ context.Context, arg db.InsertCronJobExecutionParams) (db.CronJobExecution, error) {
	return db.CronJobExecution{ID: 1, JobID: arg.JobID, Status: arg.Status}, nil
}

func (m *mockEnv) UpdateCronJobExecution(_ context.Context, arg db.UpdateCronJobExecutionParams) (db.CronJobExecution, error) {
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
func (m *mockEnv) EnqueueJob(_ context.Context, _ model.BackgroundTask) error { return nil }
func (m *mockEnv) RegisterJob(_ model.BackgroundQueueID, _ string, _ func(context.Context, []byte) error) {
}

type noopJob struct{ name string }

func (j *noopJob) Name() string                                                  { return j.name }
func (j *noopJob) Schedule() string                                              { return "0 0 * * * *" }
func (j *noopJob) ExecuteAfterStart() bool                                       { return false }
func (j *noopJob) Execute(_ context.Context, _ interface{}) (interface{}, error) { return nil, nil }

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
