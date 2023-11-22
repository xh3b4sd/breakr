package breakr

import (
	"testing"
)

func Test_Single_Interface(t *testing.T) {
	var _ Interface = NewSingle()
}
