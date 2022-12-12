package lazy

import (
	"encoding/json"
	"io"
)

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

func Default[T interface{}](def T) func(T, bool) T {
	return func(val T, ok bool) T {
		if ok {
			return val
		}
		return def
	}
}

func Require[T interface{}](ok bool, msg T) {
	if !ok {
		panic(msg)
	}
}

func JsonDecode[T interface{}](reader io.Reader) T {
	var ret T
	Assert(json.NewDecoder(reader).Decode(&ret))
	return ret
}
