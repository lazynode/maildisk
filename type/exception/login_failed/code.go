package login_failed

import "regexp"

type Type struct {
}

func CatchError(err error) {
	if regexp.MustCompile(`^LOGIN failed$`).MatchString(err.Error()) {
		panic(&Type{})
	}
	panic(err)
}
