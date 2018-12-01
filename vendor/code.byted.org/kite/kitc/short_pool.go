package kitc

import (
	"net"
	"time"

	"code.byted.org/kite/kitutil"
)

const (
	defaultConnTimeout      = 100 * time.Millisecond
	defaultConnRetryTimeout = time.Second
)

// ShortPool
type ShortPool struct {
	dialer func() Dialer
}

// NewShortPool timeout is connection timeout.
func NewShortPool() *ShortPool {
	return &ShortPool{
		dialer: func() Dialer { return nil },
	}
}

func NewShortPoolWithDialer(d func() Dialer) *ShortPool {
	return &ShortPool{
		dialer: d,
	}
}

// Get return a PoolConn instance which implemnt net.Conn interface.
func (p *ShortPool) Get(targetIns kitutil.Instance, timeout time.Duration) (net.Conn, error) {
	dial := p.dialer()
	if dial == nil {
		dial = &net.Dialer{Timeout: timeout}
	}

	ins, ok := targetIns.(*Instance)
	if !ok {
		return nil, Err("instance=%v invalid instance type", targetIns)
	}

	addr := ins.Host() + ":" + ins.Port()
	conn, err := dial.Dial("tcp", addr)
	if err != nil {
		return nil, Err("addr=%s errors=%s", addr, err)
	}
	return conn, nil
}

// Put close conn in short pool, just close the connection
func (p *ShortPool) Put(conn net.Conn, err error) error {
	if conn != nil {
		conn.Close()
	}
	return err
}
