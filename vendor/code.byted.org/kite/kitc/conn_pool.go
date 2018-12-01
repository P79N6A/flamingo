package kitc

import (
	"net"
	"sync"
	"time"

	"code.byted.org/kite/kitutil"
)

const (
	DefaultMaxIdleConnPerHost = 2
	DefaultIdleTimeout        = time.Minute
)

type ConnPool interface {
	Get(targetIns kitutil.Instance, timeout time.Duration) (net.Conn, error)
	Put(c net.Conn, err error) error
}

type PoolConn struct {
	net.Conn
	pool ConnPool
	err  error
	sync.RWMutex
}

// Close
func (c *PoolConn) Close() error {
	c.RLock()
	err := c.err
	c.RUnlock()
	return c.pool.Put(c.Conn, err)
}

func (c *PoolConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	if err != nil {
		c.Lock()
		c.err = err
		c.Unlock()
	}
	return n, err
}

func (c *PoolConn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	if err != nil {
		c.Lock()
		c.err = err
		c.Unlock()
	}
	return n, err
}

func (c *PoolConn) SetDeadline(t time.Time) error {
	err := c.Conn.SetDeadline(t)
	if err != nil {
		c.Lock()
		c.err = err
		c.Unlock()
	}
	return err
}

func (c *PoolConn) SetReadDeadline(t time.Time) error {
	err := c.Conn.SetReadDeadline(t)
	if err != nil {
		c.Lock()
		c.err = err
		c.Unlock()
	}
	return err
}

func (c *PoolConn) SetWriteDeadline(t time.Time) error {
	err := c.Conn.SetWriteDeadline(t)
	if err != nil {
		c.Lock()
		c.err = err
		c.Unlock()
	}
	return err
}

// NewPool return ShortPool or LongPool which implement PoolConn interface
func NewPool(name string, ops ...Option) ConnPool {
	opts := new(options)
	for _, do := range ops {
		do.f(opts)
	}

	if opts.useLongPool {
		maxIdle := opts.maxIdle
		if maxIdle == 0 {
			maxIdle = DefaultMaxIdleConnPerHost
		}
		maxIdleTimeout := opts.maxIdleTimeout
		if maxIdleTimeout == 0 {
			maxIdleTimeout = DefaultIdleTimeout
		}
		return &LongPool{
			peerMap:        make(map[string]*Peer),
			dialer:         func() Dialer { return nil },
			maxIdleConns:   maxIdle,
			maxIdleTimeout: maxIdleTimeout,
		}
	}
	return NewShortPool()
}
