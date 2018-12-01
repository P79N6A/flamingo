// +build !go1.7

package redis

import (
	"code.byted.org/kv/redis-v6/internal/pool"
)

type baseClient struct {
	connPool pool.Pooler
	opt      *Options

	process func(Cmder) error
	onClose func() error // hook called when client is closed
}
