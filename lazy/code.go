package lazy

func Assert(err error) {
	if err != nil {
		panic(err)
	}
}

func Unwrap[T interface{}](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}

func Catch[T interface{}](f func(T)) {
	if err := recover(); err != nil {
		exception, ok := err.(T)
		if ok {
			f(exception)
		} else {
			panic(err)
		}
	}
}
