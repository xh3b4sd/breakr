package breakr

import "time"

type Timeout struct {
	// Action is the amount of time after which the provided action will not be
	// executed anymore. Defaults to 3 seconds.
	Action time.Duration
	// Budget is the amount of attempts that can be used up when consuming the
	// timeout budget. The configured operation is being executed until Timeout
	// passed Budget times. Defaults to 1.
	Budget uint
	// Closer is the optional signal channel breaking the execution loop from
	// outside.
	Closer <-chan struct{}
	// Cooler is the optinal time to wait after any given timeout. Only takes
	// effect if Budget > 1. Defaults to -1. Disabled with -1.
	Cooler time.Duration
	// Global is the amount of time after which the breaker instance stops
	// executing the provided action and returns Passed. This is a hard global
	// timeout for each call to Breakr.Execute, applying regardless of any other
	// budget configurations. Defaults to -1. Disabled with -1.
	Global time.Duration
}
