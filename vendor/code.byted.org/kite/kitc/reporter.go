package kitc

import (
	"errors"
)

type Reporter interface {
	Report(topic, typ string, data []byte)
}

var (
	reporter Reporter

	canUseRepoter         = make(chan struct{})
	ErrRepoterExist error = errors.New("reporter is exist")
)

func SetReporter(r Reporter) error {
	if reporter != nil {
		return ErrRepoterExist
	}
	reporter = r
	close(canUseRepoter)
	return nil
}
