/*
长链接的设计：
1. 将每一个后端实例抽象成一个peer，在负载均衡的时候负责pick a peer
2. 每一个peer内部维护当前实例的状态以及一个当前实例的连接池
*/
package kitc

import (
	"bytes"
	"net"
	"sync"
	"time"

	"code.byted.org/gopkg/logs"
	"code.byted.org/kite/kitutil"
)

type Dialer interface {
	Dial(network, address string) (net.Conn, error)
}

type LongPool struct {
	// lock is used to protect peerMap
	lock sync.RWMutex
	//  peerMap store all instance address of service
	peerMap map[string]*Peer
	// dialer
	dialer         func() Dialer
	maxIdleConns   int
	maxIdleTimeout time.Duration
}

// Peer has one address, it manage all connections base on this addresss
type Peer struct {
	instance       *Instance
	addr           string
	ring           *ConnRing
	dialer         Dialer
	maxIdleConns   int
	maxIdleTimeout time.Duration
}

type PeerConn struct {
	net.Conn
	Deadline time.Time
}

// ConnRing is a struct managing all connections with same one address
type ConnRing struct {
	l    sync.Mutex
	arr  []*PeerConn
	size int
	tail int
	head int
}

// NewConnRing return a struct managing all connections with same one address
func NewConnRing(size int) *ConnRing {
	return &ConnRing{
		l:    sync.Mutex{},
		arr:  make([]*PeerConn, size+1),
		size: size,
		tail: 0,
		head: 0,
	}
}

// Push insert a PeerConn into ring
func (cr *ConnRing) Push(c *PeerConn) error {
	cr.l.Lock()
	if cr.isFull() {
		cr.l.Unlock()
		return Err("Ring is full")
	}
	cr.arr[cr.head] = c
	cr.head = cr.inc()
	cr.l.Unlock()
	return nil
}

// Pop return a PeerConn from ring, if ring is empty return nil
func (cr *ConnRing) Pop() *PeerConn {
	cr.l.Lock()
	if cr.isEmpty() {
		cr.l.Unlock()
		return nil
	}
	c := cr.arr[cr.tail]
	cr.arr[cr.tail] = nil
	cr.tail = cr.dec()
	cr.l.Unlock()
	return c
}

func (cr *ConnRing) inc() int {
	return (cr.head + 1) % (cr.size + 1)
}

func (cr *ConnRing) dec() int {
	return (cr.tail + 1) % (cr.size + 1)
}

func (cr *ConnRing) isEmpty() bool {
	return cr.tail == cr.head
}

func (cr *ConnRing) isFull() bool {
	return cr.inc() == cr.tail
}

// NewPeer create a peer which contains one address
func NewPeer(ins *Instance, maxIdle int, maxIdleTimeout time.Duration, dialer Dialer) *Peer {
	return &Peer{
		instance:       ins,
		addr:           ins.Host() + ":" + ins.Port(),
		ring:           NewConnRing(maxIdle),
		dialer:         dialer,
		maxIdleConns:   maxIdle,
		maxIdleTimeout: maxIdleTimeout,
	}
}

// Get return a net.Conn from list
func (p *Peer) Get(timeout time.Duration) (net.Conn, error) {
	// pick up connection from ring
	for {
		conn := p.ring.Pop()
		if conn == nil {
			break
		}
		if time.Now().Before(conn.Deadline) {
			return conn, nil
		}
		// close connection after deadline
		conn.Close()
	}
	var dialer Dialer
	if p.dialer != nil {
		dialer = p.dialer
	} else {
		dialer = p.getDialer(timeout)
	}
	logs.Debugf("addr: %s ring is empty and dial for new", p.addr)
	tcpConn, err := dialer.Dial("tcp", p.addr)
	if err != nil {
		return nil, err
	}
	return &PeerConn{Conn: tcpConn, Deadline: time.Now().Add(p.maxIdleTimeout)}, nil
}

func (p *Peer) Put(conn net.Conn, err error) error {
	if err != nil {
		if conn != nil {
			return conn.Close()
		}
		return nil
	}
	peerConn, ok := conn.(*PeerConn)
	if !ok {
		return conn.Close()
	}
	peerConn.Deadline = time.Now().Add(p.maxIdleTimeout)
	err = p.ring.Push(peerConn)
	if err != nil {
		return conn.Close()
	}
	return nil
}

func (p *Peer) getDialer(timeout time.Duration) Dialer {
	return &net.Dialer{Timeout: timeout}
}

// Get pick or generate a net.Conn and return
func (lp *LongPool) Get(targetIns kitutil.Instance, timeout time.Duration) (net.Conn, error) {
	internalIns, ok := targetIns.(*Instance)
	if ok == false {
		return nil, Err("kitc: Invalid Instance Type: %v", targetIns)
	}

	peer := lp.obtain(internalIns)
	conn, err := peer.Get(timeout)
	if err != nil {
		err = Err("kitc: pool get conn err: %v", err)
		return nil, err
	}
	return &PoolConn{Conn: conn, pool: lp}, nil
}

// obtain alloc a peer from longPool
func (lp *LongPool) obtain(ins *Instance) *Peer {
	var buffer bytes.Buffer
	buffer.WriteString(ins.Host())
	buffer.WriteString(":")
	buffer.WriteString(ins.Port())
	hostPort := buffer.String()
	lp.lock.RLock()
	peer, ok := lp.peerMap[hostPort]
	lp.lock.RUnlock()
	if ok {
		return peer
	}
	peer = NewPeer(ins, lp.maxIdleConns, lp.maxIdleTimeout, lp.dialer())
	lp.lock.Lock()
	lp.peerMap[hostPort] = peer
	lp.lock.Unlock()
	return peer
}

// Put back a connection to longPool
func (lp *LongPool) Put(conn net.Conn, err error) error {
	if conn == nil {
		return nil
	}
	hostPort := conn.RemoteAddr().String()
	lp.lock.RLock()
	peer, ok := lp.peerMap[hostPort]
	lp.lock.RUnlock()
	if !ok {
		return conn.Close()
	}
	return peer.Put(conn, err)
}
