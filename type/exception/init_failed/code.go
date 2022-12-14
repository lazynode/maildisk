package init_failed

type Type struct {
	Error error
}

func CatchError(err error) {
	panic(&Type{err})
}
