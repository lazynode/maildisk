package mail_box_already_exists

import "regexp"

type Type struct {
}

func CatchError(err error) {
	if regexp.MustCompile(`^CREATE failed: mailbox already exists$`).MatchString(err.Error()) {
		panic(&Type{})
	}
	panic(err)
}
