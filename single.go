package breakr

import (
	"github.com/xh3b4sd/tracer"
)

// Single executes the given operation exactly once, regardless if it fails or
// succeeds. It is usually only used for testing.
type Single struct{}

func NewSingle() *Single {
	return &Single{}
}

func (s *Single) Execute(act func() error) error {
	err := s.Wrapper(act)()
	if err != nil {
		return tracer.Mask(err)
	}

	return nil
}

func (s *Single) Wrapper(act func() error) func() error {
	return func() error {
		err := act()
		if err != nil {
			return tracer.Mask(err)
		}

		return nil
	}
}
