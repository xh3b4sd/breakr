package breakr

import (
	"errors"

	"github.com/xh3b4sd/tracer"
)

var Cancel = &tracer.Error{
	Kind: "cancel",
	Desc: "Cancel is the error returned by actions in order to break the execution loop. Actions returning Cancel force any budget implementation to stop any further executions.",
}

func IsCancel(err error) bool {
	return errors.Is(err, Cancel)
}

var Closed = &tracer.Error{
	Kind: "closed",
	Desc: "Closed is the error returned by budget implementations if the configured signal channel got closed. This might be done in order to break the execution loop from a process point of view. Processes receiving kill signals may forward shutdown signals to budget implementations in order to force any budget implementation to stop any further executions.",
}

func IsClosed(err error) bool {
	return errors.Is(err, Closed)
}

var Filled = &tracer.Error{
	Kind: "filled",
	Desc: "Filled is the error returned by budget implementations if the configured limiter queue is full. This may happen if the configured action was tried to be executed too many times within a given time window.",
}

func IsFilled(err error) bool {
	return errors.Is(err, Filled)
}

var Passed = &tracer.Error{
	Kind: "passed",
	Desc: "Passed is the error returned by budget implementations if the configured timeout expired. Timeouts may apply to individual executions of the configured action or globally for a speficic execution of Breakr.Execute.",
}

func IsPassed(err error) bool {
	return errors.Is(err, Passed)
}

var Repeat = &tracer.Error{
	Kind: "repeat",
	Desc: "Repeat is the error returned by actions in order to repeat the execution loop early without taking away from any failure or success budget.",
}

func IsRepeat(err error) bool {
	return errors.Is(err, Repeat)
}
