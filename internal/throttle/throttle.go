// Package throttle coalesces a stream of signals into at most one run per
// interval. The first signal schedules a run one interval later, signals that
// arrive inside that window join it, and a signal that arrives while a run is
// in progress schedules the next one — so a continuous stream keeps producing
// runs at the interval, and the last signal is always followed by a run.
package throttle

import "time"

type Throttle struct {
	interval time.Duration
	run      func()
	signals  chan struct{}
	sleep    func(time.Duration)
}

func New(interval time.Duration, run func()) *Throttle {
	return start(interval, run, time.Sleep)
}

func start(interval time.Duration, run func(), sleep func(time.Duration)) *Throttle {
	t := &Throttle{
		interval: interval,
		run:      run,
		signals:  make(chan struct{}, 1),
		sleep:    sleep,
	}
	go t.loop()
	return t
}

// Signal asks for a run. It never blocks: a signal sent while one is already
// pending joins the pending run.
func (t *Throttle) Signal() {
	select {
	case t.signals <- struct{}{}:
	default:
	}
}

func (t *Throttle) loop() {
	for range t.signals {
		t.sleep(t.interval)
		select {
		case <-t.signals:
		default:
		}
		t.run()
	}
}
