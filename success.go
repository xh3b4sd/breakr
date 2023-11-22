package breakr

type Success struct {
	// Budget is the required amount of successful executions of the provided
	// action. Breakr.Execute guarantees act to be executed Budget times without
	// any error returned, unless the configured signal channel got closed or the
	// configured timeout passed. Defaults to 1.
	Budget uint
}
