package lazy

import (
	"encoding/json"
	"io"
	"sync"
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

func JsonDecodePtr[T interface{}](reader io.Reader) *T {
	var ret T
	Assert(json.NewDecoder(reader).Decode(&ret))
	return &ret
}

func Array[T interface{}](val ...T) []T {
	return val
}

func Flatten[T interface{}](val ...[]T) []T {
	ret := make([]T, 0)
	for _, v := range val {
		ret = append(ret, v...)
	}
	return ret
}

func ParallelReturn[T interface{}](fs ...func() T) []T {
	ret := make([]T, len(fs))
	var wg sync.WaitGroup
	wg.Add(len(fs))
	for i := range fs {
		go func(i int) {
			defer wg.Done()
			ret[i] = fs[i]()
		}(i)
	}
	wg.Wait()
	return ret
}
