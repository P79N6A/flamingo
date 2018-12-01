package kitc

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"code.byted.org/kite/kitutil"
)

const (
	IDX = 0x3ff

	DefaultLimitRetryRate = 0.2
	DefaultMaxRetryTimes  = 0
)

type LimitRetrier struct {
	needRetry func(ret interface{}, err error) bool
	failCnt   int32
	idx       uint32
	samples   [1024]int32
}

func NewLimitRetrier(needed func(ret interface{}, err error) bool) *LimitRetrier {
	return &LimitRetrier{
		failCnt:   0,
		samples:   [1024]int32{},
		idx:       0,
		needRetry: needed,
	}
}

func (lr *LimitRetrier) Do(limitRate float64, maxRetry int, f func() (interface{}, error)) (interface{}, error) {
	resp, err := f()
	lr.recordNormal()
	cnt := 0
	for lr.needRetry(resp, err) {
		if cnt >= maxRetry {
			// TODO(xiangchao.01): add this message into err
			break
		}

		if atomic.LoadInt32(&lr.failCnt) > int32(1024*limitRate) {
			// TODO(xiangchao.01): add this message into err
			break
		}
		resp, err = f()
		lr.recoredRetry()
		cnt++
	}
	return resp, err
}

func (lr *LimitRetrier) recordNormal() {
	idx := atomic.AddUint32(&lr.idx, 1)
	idx = idx & IDX
	if atomic.CompareAndSwapInt32(&lr.samples[idx], 1, 0) {
		atomic.AddInt32(&lr.failCnt, -1)
	}
}

func (lr *LimitRetrier) recoredRetry() {
	idx := atomic.AddUint32(&lr.idx, 1)
	idx = idx & IDX
	if atomic.CompareAndSwapInt32(&lr.samples[idx], 0, 1) {
		atomic.AddInt32(&lr.failCnt, 1)
	}
}

type RetryManager struct {
	l           sync.RWMutex
	m           map[string]*LimitRetrier
	wantedRetry func(ret interface{}, err error) bool
}

func NewRetryManager(f func(ret interface{}, err error) bool) *RetryManager {
	return &RetryManager{
		l:           sync.RWMutex{},
		m:           make(map[string]*LimitRetrier),
		wantedRetry: f,
	}
}

func (rm *RetryManager) RetryKey(ctx context.Context) (string, error) {
	callTuple, ok := kitutil.GetCtxCallTuple(ctx)
	if !ok {
		return "", ErrNoCallTuple
	}
	if callTuple.To == "" {
		return "", ErrNoTargetService
	}
	if callTuple.Method == "" {
		return "", ErrNoTargetMethod
	}
	key := fmt.Sprintf("%s:%s:%s", callTuple.To, callTuple.ToCluster, callTuple.Method)
	return key, nil
}

func (rm *RetryManager) Retry(key string) *LimitRetrier {
	rm.l.RLock()
	retrier, ok := rm.m[key]
	rm.l.RUnlock()
	if !ok {
		retrier = NewLimitRetrier(rm.wantedRetry)
		rm.l.Lock()
		rm.m[key] = retrier
		rm.l.Unlock()
	}
	return retrier
}
