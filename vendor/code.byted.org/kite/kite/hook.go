package kite

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"

	"code.byted.org/gopkg/asyncache"
	"code.byted.org/gopkg/etcd_util"
	etcdClient "code.byted.org/gopkg/etcd_util/client"
	"code.byted.org/gopkg/tokenbucket"
)

const (
	panicFmt string = "service.thrift.%s.panic"
	connsFmt string = "service.thrift.%s.conns"

	etcdLimitQPS  string = "/kite/limit/qps/%s"
	etcdLimitConn string = "/kite/limit/conn/%s"
)

var (
	limitQPS      int64
	limitMaxConns int64
	currentConns  int64
	LimitErr      = errors.New("LimitErr")
)

type Limiter interface {
	LimitQPS() (int, error)
	LimitConns() (int, error)
}

type etcdLimiter struct {
	limitCache    *asyncache.SingleAsyncCache
	keyLimitQPS   string
	keyLimitConns string
}

func NewEtcdLimiter(sn string) *etcdLimiter {
	var f func(key string) (interface{}, error)
	etcdC, err := etcdutil.GetDefaultClient()
	if err != nil {
		f = func(key string) (interface{}, error) {
			return 0, nil
		}
	} else {
		f = func(key string) (interface{}, error) {
			val, err := etcdC.Get(context.Background(), key, nil)
			if err != nil {
				if etcdClient.IsKeyNotFound(err) {
					return 0, nil
				}
				return 0, err
			}
			intV, err := strconv.ParseInt(val.Node.Value, 10, 64)
			if err != nil {
				return 0, err
			}
			return int(intV), nil
		}
	}
	return &etcdLimiter{
		limitCache:    asyncache.NewSingleAsyncCache(f),
		keyLimitQPS:   fmt.Sprintf(etcdLimitQPS, sn),
		keyLimitConns: fmt.Sprintf(etcdLimitConn, sn),
	}
}

// LimitQPS default is 0
func (el *etcdLimiter) LimitQPS() (int, error) {
	val, err := el.limitCache.Get(el.keyLimitQPS)
	if err != nil {
		return 0, err
	}
	return val.(int), nil
}

// LimitConns default is 0
func (el *etcdLimiter) LimitConns() (int, error) {
	val, err := el.limitCache.Get(el.keyLimitConns)
	if err != nil {
		return 0, err
	}
	return val.(int), nil
}

func CurrentConns() int64 {
	return atomic.LoadInt64(&currentConns)
}

func NoOp() {}

type PanicHooker struct {
	panicFmt    string
	serviceName string
	metricsKey  string
}

func NewPanicHooker(pf, sn string) *PanicHooker {
	return &PanicHooker{
		panicFmt:    pf,
		serviceName: sn,
		metricsKey:  fmt.Sprintf(pf, sn),
	}
}

func (ph *PanicHooker) OnPanic() {
	metricsClient.EmitCounter(ph.metricsKey, 1, "", nil)
}

type LimitHooker struct {
	limitMaxConn int32
	limitQPS     int32
	metricsKey   string
	serviceName  string
	limiter      Limiter
	lock         *sync.RWMutex
	protectQPS   *tokenbucket.TokenBucket
}

func NewLimitHooker(limitQPS, limitMaxConn int, limiter Limiter, sn string) *LimitHooker {
	ret := &LimitHooker{
		limiter:      limiter,
		limitMaxConn: int32(limitMaxConn),
		limitQPS:     int32(limitQPS),
		metricsKey:   fmt.Sprintf(connsFmt, sn),
		serviceName:  sn,
		lock:         new(sync.RWMutex),
		protectQPS:   tokenbucket.New(10, int(limitQPS), 2*int(limitQPS)),
	}
	return ret
}

func (ph *LimitHooker) LimitQPS() int {
	return int(atomic.LoadInt32(&ph.limitQPS))
}

func (ph *LimitHooker) LimitMaxConns() int {
	return int(atomic.LoadInt32(&ph.limitMaxConn))
}

func (ph *LimitHooker) OnProcess() error {
	oldQPS := int(atomic.LoadInt32(&ph.limitQPS))
	newQPS, err := ph.limiter.LimitQPS()
	if err == nil {
		if newQPS > 0 && oldQPS != newQPS {
			if atomic.CompareAndSwapInt32(&ph.limitQPS, int32(oldQPS), int32(newQPS)) {
				ph.lock.Lock()
				ph.protectQPS = tokenbucket.New(10, newQPS, 2*newQPS)
				ph.lock.Unlock()
			}
		}
	}
	ph.lock.RLock()
	pQPS := ph.protectQPS
	ph.lock.RUnlock()

	if pQPS.Consume(ph.serviceName) {
		return nil
	}
	return fmt.Errorf("over limit QPS=%d", atomic.LoadInt32(&ph.limitQPS))
}

func (ph *LimitHooker) OnConnect() error {
	val, err := ph.limiter.LimitConns()
	if err == nil {
		if val > 0 && val != int(atomic.LoadInt32(&ph.limitMaxConn)) {
			atomic.StoreInt32(&ph.limitMaxConn, int32(val))
		}
	}
	curConn := atomic.AddInt64(&currentConns, 1)
	if limit := atomic.LoadInt32(&ph.limitMaxConn); int32(curConn) > limit {
		atomic.AddInt64(&currentConns, -1)
		return fmt.Errorf("current=%d limit=%d connections overhead", curConn, limit)
	}
	metricsClient.EmitStore(ph.metricsKey, atomic.LoadInt64(&currentConns), "", nil)
	return nil
}

func (ph *LimitHooker) OnDisconnect() {
	atomic.AddInt64(&currentConns, -1)
	metricsClient.EmitStore(ph.metricsKey, atomic.LoadInt64(&currentConns), "", nil)
}
