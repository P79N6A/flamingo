package namekeeper

import (
	"errors"
	"log"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"code.byted.org/gopkg/pkg/compress/snappy"
	"code.byted.org/gopkg/pkg/net2"
	"code.byted.org/gopkg/pkg/sync/singleflight"

	"code.byted.org/gopkg/naming/msgtype"
)

var (
	LocalNamekeeperUDPAddr = "127.0.0.1:2012"
	NamekeeperAPIDomain    = "namekeeper-api.byted.org"
	NamekeeperPort         = "2012"
)

const (
	maxIdleConn   = 3
	namekeeperPSM = "toutiao.naming.namekeeper"
)

var (
	errClosed     = errors.New("namekeeper client closed")
	errNoInstance = errors.New("no instance available")
)

type cacheService struct {
	t  time.Time
	ii *msgtype.ServiceInstaces
}

type Namekeeper struct {
	mu      sync.Mutex
	closed  bool
	timeout time.Duration
	addrs   []*net.UDPAddr
	conns   []net.PacketConn

	cmu   sync.RWMutex
	cache map[string]*cacheService // service+cluster as key

	singleFlight singleflight.Group

	reqn uint64
}

var (
	mu          sync.RWMutex
	gNamekeeper *Namekeeper
)

func GetDefaultNamekeeper() (*Namekeeper, error) {
	mu.RLock()
	nk := gNamekeeper
	mu.RUnlock()
	if nk != nil {
		return nk, nil
	}
	mu.Lock()
	defer mu.Unlock()
	if gNamekeeper == nil {
		nk, err := newDefaultNamekeeper()
		if err != nil {
			return nil, err
		}
		gNamekeeper = nk
		return nk, nil
	}
	return gNamekeeper, nil
}

func getNamekeeperAddrs(local bool) ([]string, error) {
	if local {
		k, _ := NewNamekeeper([]string{LocalNamekeeperUDPAddr}, 100*time.Millisecond)
		if ii, err := k.Get(namekeeperPSM); err == nil {
			k.Close()
			return ii.Addrs(), nil
		}
	}
	ss, err := net2.LookupIPAddr(NamekeeperAPIDomain, 24*time.Hour)
	if err != nil {
		return nil, err
	}
	addrs := make([]string, len(ss))
	for i := range rand.Perm(len(addrs)) {
		addrs[i] = ss[i] + ":" + NamekeeperPort
	}
	k, er := NewNamekeeper(addrs, 300*time.Millisecond)
	if er != nil {
		return nil, er
	}
	if ii, err := k.Get(namekeeperPSM); err == nil {
		k.Close()
		return ii.Addrs(), nil
	} else {
		return nil, err
	}
}

func newDefaultNamekeeper() (*Namekeeper, error) {
	addrs, err := getNamekeeperAddrs(false)
	if err != nil {
		return nil, err
	}
	k, err := NewNamekeeper(addrs, time.Second)
	if err != nil {
		return nil, err
	}
	go func() {
		for _ = range time.Tick(3 * time.Minute) {
			if err := k.renewAddrs(); err != nil {
				if err == errClosed {
					break
				}
				log.Println("Namekeeper: renew err %s", err)
			}
		}
	}()
	return k, nil
}

func resolveUDPAddrs(addrs []string) []*net.UDPAddr {
	var aa []*net.UDPAddr
	for _, s := range addrs {
		a, err := net.ResolveUDPAddr("udp", s)
		if err != nil {
			continue
		}
		aa = append(aa, a)
	}
	return aa
}

func NewNamekeeper(addrs []string, timeout time.Duration) (*Namekeeper, error) {
	aa := resolveUDPAddrs(addrs)
	if len(aa) == 0 {
		return nil, errors.New("no namekeeper avaiable")
	}
	cache := make(map[string]*cacheService)
	return &Namekeeper{addrs: aa, timeout: timeout, cache: cache}, nil
}

func (k *Namekeeper) returnInstaces(ii *msgtype.ServiceInstaces, err error, q options) (*msgtype.ServiceInstaces, error) {
	if ii == nil && q.stale {
		k.cmu.RLock()
		item := k.cache[q.CacheKey()]
		k.cmu.RUnlock()
		if item != nil {
			err = nil
			ii = item.ii
		}
	}
	if err != nil {
		return nil, err
	}
	if ii == nil || len(ii.Instances) == 0 {
		return nil, errNoInstance
	}
	if q.SingleShot && len(ii.Instances) > 1 {
		ret := *ii
		var weight int
		for _, s := range ret.Instances {
			weight += int(s.Weight)
		}
		weight -= rand.Intn(weight)
		for i, s := range ret.Instances {
			weight -= int(s.Weight)
			if weight <= 0 {
				ret.Instances = ret.Instances[i : i+1]
				break
			}
		}
		return &ret, nil
	}
	return ii, nil
}

func (k *Namekeeper) Get(name string, ops ...GetOption) (*msgtype.ServiceInstaces, error) {
	var q options
	q.stale = true
	q.cache = time.Second
	q.Limit = 1000
	q.Op = msgtype.OpQuery
	q.Service = name
	for _, op := range ops {
		op(&q)
	}
	if q.RequestID == "" {
		q.RequestID = strconv.FormatUint(rand.Uint64(), 16)
	}

	if q.cache > 0 {
		key := q.CacheKey()
		k.cmu.RLock()
		item := k.cache[key]
		k.cmu.RUnlock()
		if item != nil && time.Since(item.t) < q.cache {
			return k.returnInstaces(item.ii, nil, q)
		}
	}

	_q := q
	_q.SingleShot = false // we implements singleshot in `returnInstaces`

	if q.cache > 0 {
		key := q.CacheKey()
		fn := func() (interface{}, error) {
			ii, err := k.sendreq(_q)
			k.cmu.Lock()
			k.cache[key] = &cacheService{t: time.Now(), ii: ii}
			k.cmu.Unlock()
			return ii, err
		}
		ret, err, _ := k.singleFlight.Do(key, fn)
		ii := ret.(*msgtype.ServiceInstaces)
		return k.returnInstaces(ii, err, q)
	}
	ii, err := k.sendreq(_q)
	return k.returnInstaces(ii, err, q)
}

func (k *Namekeeper) Close() {
	k.mu.Lock()
	for _, conn := range k.conns {
		conn.Close()
	}
	k.conns = nil
	k.closed = true
	k.mu.Unlock()
}

func (k *Namekeeper) renewAddrs() error {
	var ss []string
	ii, err := k.Get(namekeeperPSM)
	if ii != nil {
		for _, e := range ii.Instances {
			ss = append(ss, e.Addr)
		}
	} else {
		ss, err = getNamekeeperAddrs(true)
	}
	if err != nil {
		return err
	}
	addrs := resolveUDPAddrs(ss)
	if len(addrs) == 0 {
		return errors.New("no namekeeper avaiable")
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.closed {
		return errClosed
	}
	k.addrs = addrs
	return nil
}

func (k *Namekeeper) getconn() (net.PacketConn, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.closed {
		return nil, errClosed
	}
	if len(k.conns) > 0 {
		conn := k.conns[len(k.conns)-1]
		k.conns = k.conns[0 : len(k.conns)-1]
		return conn, nil
	}
	return net.ListenPacket("udp", ":0")
}

func (k *Namekeeper) putconn(conn net.PacketConn) {
	k.mu.Lock()
	if len(k.conns) > maxIdleConn {
		conn.Close()
	} else {
		k.conns = append(k.conns, conn)
	}
	k.mu.Unlock()
}

var udpbuf = sync.Pool{
	New: func() interface{} {
		return make([]byte, 65<<10, 65<<10)
	},
}

func (k *Namekeeper) GetRequestCount() uint64 {
	return atomic.LoadUint64(&k.reqn)
}

func (k *Namekeeper) sendreq(q options) (*msgtype.ServiceInstaces, error) {
	atomic.AddUint64(&k.reqn, 1)

	var n int
	var err error

	timeout := k.timeout
	if q.timeout != 0 {
		timeout = q.timeout
	}
	addrs := k.getaddrs()
	if len(addrs) == 0 {
		return nil, errClosed
	}

	conn, err := k.getconn()
	if err != nil {
		return nil, err
	}

	b := udpbuf.Get().([]byte)
	defer func() {
		udpbuf.Put(b)
		k.putconn(conn)
	}()

	var m *msgtype.ServiceInstaces

ADDRS_LOOP:
	for i := 0; i < len(addrs); i++ {
		d := timeout / time.Duration(len(addrs))
		if d < 50*time.Millisecond {
			d = 50 * time.Millisecond
		}
		conn.SetDeadline(time.Now().Add(d))

		data, _ := q.MarshalMsg(b[:0])
		if _, err = conn.WriteTo(data, addrs[i]); err != nil {
			continue
		}
		for {
			n, _, err = conn.ReadFrom(b)
			if err != nil {
				continue ADDRS_LOOP
			}
			data = b[:n]
			if !q.NoCompress {
				n, err := snappy.DecodedLen(data)
				if err != nil {
					continue
				}
				dest := make([]byte, n)
				if _, err := snappy.Decode(dest, data); err != nil {
					continue
				}
				data = dest
			}
			m = &msgtype.ServiceInstaces{}
			_, err = m.UnmarshalMsg(data)
			if err != nil {
				continue
			}
			if m.Service != q.Service || m.RequestID != q.RequestID {
				continue
			}
			break ADDRS_LOOP
		}
	}
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (k *Namekeeper) getaddrs() []*net.UDPAddr {
	k.mu.Lock()
	addrs := k.addrs
	k.mu.Unlock()
	return addrs
}

func udpAddrEqual(a, b *net.UDPAddr) bool {
	if a == nil || b == nil {
		return false
	}
	if a.IP.Equal(b.IP) && a.Port == b.Port {
		return true
	}
	return false
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
