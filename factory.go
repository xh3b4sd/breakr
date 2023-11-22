package breakr

func Default() Interface {
	return New(Config{})
}

func Fake() Interface {
	return NewSingle()
}
