package tos

import (
	"sync"
)

type LimitCurrent struct {
	wg    sync.WaitGroup
	limit chan struct{}
}

func NewLimitCurrent(l int) *LimitCurrent {
	return &LimitCurrent{
		wg:    sync.WaitGroup{},
		limit: make(chan struct{}, l),
	}
}

func (l *LimitCurrent) Do(f func()) {
	l.limit <- struct{}{}
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		f()
		<-l.limit
	}()
}
func (l *LimitCurrent) Wait() {
	l.wg.Wait()
}
