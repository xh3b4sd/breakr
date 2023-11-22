package breakr

import (
	"sync"
	"time"

	"github.com/xh3b4sd/tracer"
)

type Limiter struct {
	// Budget is the maximum amount of actions allowed to be queued at the same
	// time. Budget set to 3 would cause Execute to return breakr.Filled after the
	// 4th invocation, considering that 3 actions would be executing already.
	// Defaults to 3.
	Budget uint
	// Cooler is the optinal time to wait after action execution that it takes to
	// drain a task from the action queue. Cooler defines a time window in which a
	// maximum number of concurrent actions can be executed as defined by Budget.
	// Cooler set to 500ms would cause Execute to only allow execution of 3
	// actions within 500ms, given the default configuration of Budget at 3.
	// Defaults to -1. Disabled with -1.
	Cooler time.Duration
}

func (l *Limiter) New() *limiter {
	return &limiter{
		bud: make(chan struct{}, l.Budget),
		coo: l.Cooler,
		que: make(chan struct{}, l.Budget),
	}
}

type limiter struct {
	bud chan struct{}
	coo time.Duration
	mut sync.Mutex
	que chan struct{}
	tim []time.Time
}

func (l *limiter) Execute(act func() error) error {
	var dur time.Duration
	var tim time.Time
	{
		l.mut.Lock()
		if len(l.tim) != 0 {
			tim = l.tim[0]
			dur = time.Since(tim)
		}
		l.mut.Unlock()
	}

	{
		if len(l.bud) == cap(l.bud) && dur >= l.coo {
			{
				l.mut.Lock()
			}

			{
				for _, t := range l.tim {
					if t.Add(l.coo).Before(time.Now().UTC()) {
						{
							<-l.bud
						}

						{
							l.tim = l.tim[1:]
						}
					}
				}
			}

			{
				l.mut.Unlock()
			}
		}
	}

	{
		if l.coo != -1 && !tim.IsZero() && len(l.bud) == cap(l.bud) {
			return tracer.Maskf(Filled, "actions throttled for another %s", l.coo-dur)
		}
	}

	{
		if len(l.que) == cap(l.que) {
			return tracer.Maskf(Filled, "%d actions already queued", len(l.que))
		}
	}

	{
		l.mut.Lock()
		l.tim = append(l.tim, time.Now().UTC())
		l.mut.Unlock()
	}

	{
		l.bud <- struct{}{}
		l.que <- struct{}{}
	}

	{
		defer func() { <-l.que }()
	}

	return act()
}
