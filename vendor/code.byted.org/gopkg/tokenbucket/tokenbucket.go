package tokenbucket

import (
	"container/list"
	"sync"
	"time"
)

type Item struct {
	key    string
	t      int64
	tokens int
}

type TokenBucket struct {
	size     int
	rate     int
	capacity int
	data     map[string]*list.Element
	l        *list.List
	lock     *sync.RWMutex
}

func New(size, rate, capacity int) *TokenBucket {
	return &TokenBucket{
		size,
		rate,
		capacity,
		make(map[string]*list.Element),
		list.New(),
		new(sync.RWMutex),
	}
}

func minInt(a, b int) int {
	if a > b {
		return b
	}
	return a
}

func (tb *TokenBucket) Consume(key string) bool {
	tb.lock.Lock()
	defer tb.lock.Unlock()

	var tokens int

	if value, ok := tb.data[key]; ok {
		tb.l.MoveToFront(value)
		now := time.Now().Unix()
		if value.Value.(*Item).t == now {
			tokens = value.Value.(*Item).tokens
			if tokens <= 0 {
				return false
			}
			value.Value.(*Item).tokens = tokens - 1
			return true
		}

		delta := int(now-value.Value.(*Item).t) * tb.rate
		tokens = minInt(value.Value.(*Item).tokens+delta, tb.capacity)
		value.Value.(*Item).t = now
		value.Value.(*Item).tokens = tokens - 1
		return true
	}
	if len(tb.data) >= tb.size {
		delete(tb.data, tb.l.Back().Value.(*Item).key)
		tb.l.Remove(tb.l.Back())
	}

	tb.data[key] = tb.l.PushFront(&Item{key, time.Now().Unix(), tb.rate - 1})
	return true
}

type BlockTokenBucket struct {
	rate     int
	capacity int
	bucket   chan struct{}
}

func NewBlockTokenBucket(rate, capacity int) *BlockTokenBucket {
	if rate <= 0 || rate > 1e9 {
		panic("invalid BlockTokenBucket rate")
	}
	bt := &BlockTokenBucket{
		bucket:   make(chan struct{}, capacity),
		rate:     rate,
		capacity: capacity,
	}
	go bt.runLoop()
	return bt
}

func (bt *BlockTokenBucket) runLoop() {
	duration := 1e9 / bt.rate
	for {
		time.Sleep(time.Duration(duration) * time.Nanosecond)
		bt.bucket <- struct{}{}
	}
}

func (bt *BlockTokenBucket) Consume() {
	<-bt.bucket
	return
}
