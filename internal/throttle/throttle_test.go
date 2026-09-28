package throttle

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// harness replaces the clock with two channels: entered reports that the loop
// began waiting out an interval, release ends that wait.
type harness struct {
	entered chan struct{}
	release chan struct{}
	started chan struct{}
	finish  chan struct{}
	runs    int
}

func newHarness(t *testing.T) (*Throttle, *harness) {
	t.Helper()
	h := &harness{
		entered: make(chan struct{}),
		release: make(chan struct{}),
		started: make(chan struct{}),
		finish:  make(chan struct{}),
	}
	sleep := func(time.Duration) {
		h.entered <- struct{}{}
		<-h.release
	}
	run := func() {
		h.runs++
		h.started <- struct{}{}
		<-h.finish
	}
	return start(time.Hour, run, sleep), h
}

func (h *harness) waitInterval(t *testing.T) {
	t.Helper()
	select {
	case <-h.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("throttle never started waiting out the interval")
	}
}

func (h *harness) endInterval() {
	h.release <- struct{}{}
}

func (h *harness) waitRun(t *testing.T) {
	t.Helper()
	select {
	case <-h.started:
	case <-time.After(5 * time.Second):
		t.Fatal("throttle never ran")
	}
}

func (h *harness) endRun() {
	h.finish <- struct{}{}
}

func (h *harness) requireIdle(t *testing.T) {
	t.Helper()
	select {
	case <-h.entered:
		t.Fatal("throttle scheduled a run without a signal")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestSignalsInsideTheWindowCoalesce(t *testing.T) {
	th, h := newHarness(t)

	th.Signal()
	h.waitInterval(t)
	for range 50 {
		th.Signal()
	}
	h.endInterval()
	h.waitRun(t)
	h.endRun()

	h.requireIdle(t)
	require.Equal(t, 1, h.runs)
}

func TestSignalDuringRunSchedulesTrailingRun(t *testing.T) {
	th, h := newHarness(t)

	th.Signal()
	h.waitInterval(t)
	h.endInterval()
	h.waitRun(t)
	th.Signal()
	h.endRun()

	h.waitInterval(t)
	h.endInterval()
	h.waitRun(t)
	h.endRun()

	h.requireIdle(t)
	require.Equal(t, 2, h.runs)
}

func TestContinuousStreamRunsEveryInterval(t *testing.T) {
	th, h := newHarness(t)

	stop := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		for {
			select {
			case <-stop:
				return
			default:
				th.Signal()
			}
		}
	}()

	for range 5 {
		h.waitInterval(t)
		h.endInterval()
		h.waitRun(t)
		h.endRun()
	}
	close(stop)
	<-stopped

	require.Equal(t, 5, h.runs)
}

func TestSignalNeverBlocks(t *testing.T) {
	th := start(time.Hour, func() {}, func(time.Duration) { select {} })

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 1000 {
			th.Signal()
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Signal blocked with a run already pending")
	}
}
