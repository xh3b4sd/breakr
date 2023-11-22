package breakr

import (
	"time"

	"github.com/xh3b4sd/tracer"
)

type Config struct {
	Failure Failure
	Limiter Limiter
	Success Success
	Timeout Timeout
}

type Breakr struct {
	fai Failure
	lim *limiter
	suc Success
	tim Timeout
}

func New(config Config) *Breakr {
	{
		if config.Failure.Budget == 0 {
			config.Failure.Budget = 3
		}
		if config.Failure.Cooler == 0 {
			config.Failure.Cooler = 1 * time.Second
		}
	}

	{
		if config.Success.Budget == 0 {
			config.Success.Budget = 1
		}
	}

	{
		if config.Limiter.Budget == 0 {
			config.Limiter.Budget = 3
		}
		if config.Limiter.Cooler == 0 {
			config.Limiter.Cooler = -1
		}
	}

	{
		if config.Timeout.Action == 0 {
			config.Timeout.Action = 3 * time.Second
		}
		if config.Timeout.Budget == 0 {
			config.Timeout.Budget = 1
		}
		if config.Timeout.Cooler == 0 {
			config.Timeout.Cooler = -1
		}
		if config.Timeout.Global == 0 {
			config.Timeout.Global = -1
		}
	}

	b := &Breakr{
		fai: config.Failure,
		lim: config.Limiter.New(),
		suc: config.Success,
		tim: config.Timeout,
	}

	return b
}

func (b *Breakr) Execute(act func() error) error {
	err := b.Wrapper(act)()
	if err != nil {
		return tracer.Mask(err)
	}

	return nil
}

func (b *Breakr) Wrapper(act func() error) func() error {
	return func() error {
		var fco uint
		var sco uint
		var tco uint

		erc := make(chan error, 1)
		exe := make(chan struct{}, 1)
		glo := timeout(b.tim.Global)
		suc := make(chan struct{}, 1)

		exe <- struct{}{}

		for {
			select {
			case <-exe:
				go func() {
					err := b.lim.Execute(act)
					if err != nil {
						erc <- tracer.Mask(err)
					} else {
						suc <- struct{}{}
					}
				}()
			case <-suc:
				sco++
				if sco >= b.suc.Budget {
					return nil
				}

				exe <- struct{}{}
			case <-b.tim.Closer:
				return tracer.Mask(Closed)
			case <-glo:
				return tracer.Mask(Passed)
			case <-timeout(b.tim.Action):
				tco++

				if tco >= b.tim.Budget {
					return tracer.Mask(Passed)
				}

				if b.tim.Cooler != -1 {
					time.Sleep(b.tim.Cooler)
				}

				exe <- struct{}{}
			case err := <-erc:
				if IsCancel(err) {
					return tracer.Mask(err)
				}

				if IsFilled(err) {
					return tracer.Mask(err)
				}

				if IsRepeat(err) {
					// fall through
				} else {
					fco++
					if fco >= b.fai.Budget {
						return tracer.Mask(err)
					}

					if b.fai.Cooler != -1 {
						time.Sleep(b.fai.Cooler)
					}
				}

				exe <- struct{}{}
			}
		}
	}
}

func timeout(dur time.Duration) <-chan time.Time {
	if dur != -1 {
		return time.After(dur)
	}

	return nil
}
