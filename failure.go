package breakr

import "time"

type Failure struct {
	// Budget is the amount of attempts that can be used up when consuming the
	// error budget. The configured operation is being executed until it succeeds
	// or the error budget is used up. A budget of 3 means the configured
	// operation will be executed up to 3 times. Defaults to 3.
	Budget uint
	// Cooler is the optinal time to wait after any given retry. Given a Cooler of
	// 5 seconds and a Budget of 3 the execution would happen as follows. Defaults
	// to 1s. Disabled with -1.
	//
	//     * first execution fails
	//     * wait 5 seconds
	//     * second execution fails
	//     * wait 5 seconds
	//     * third execution fails
	//     * return error
	//
	Cooler time.Duration
}
