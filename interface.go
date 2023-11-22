package breakr

type Interface interface {
	Execute(act func() error) error
	Wrapper(act func() error) func() error
}
