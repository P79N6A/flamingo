package namekeeper

import (
	"time"

	"code.byted.org/gopkg/naming/msgtype"
)

type options struct {
	msgtype.Request
	timeout time.Duration
	cache   time.Duration
	stale   bool
}

type GetOption func(q *options)

func WithCluster(cluster string) GetOption {
	return func(op *options) {
		op.Cluster = cluster
	}
}

func WithLimit(n uint16) GetOption {
	return func(op *options) {
		op.Limit = n
	}
}

func WithSingleShot() GetOption {
	return func(op *options) {
		op.SingleShot = true
	}
}

func WithRequestID(id string) GetOption {
	return func(q *options) {
		q.RequestID = id
	}
}

func WithTimeout(timeout time.Duration) GetOption {
	return func(q *options) {
		q.timeout = timeout
	}
}

func WithCache(timeout time.Duration) GetOption {
	return func(q *options) {
		q.cache = timeout
	}
}

func WithCacheStale(b bool) GetOption {
	return func(q *options) {
		q.stale = b
	}
}

func (q *options) CacheKey() string {
	return q.Service + "|" + q.Cluster
}
