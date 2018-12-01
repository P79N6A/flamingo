package kitc

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Option struct {
	f func(*options)
}

type CircuitbreakerOption struct {
	ErrorRate      float64
	MinSamples     int
	MaxConcurrency int
}

type options struct {
	readWriteTimeout time.Duration
	connTimeout      time.Duration
	connMaxRetryTime time.Duration
	cluster          string
	idc              string
	insList          []*Instance
	iks              IKService
	useWatcher       bool
	useLongPool      bool
	maxIdle          int
	maxIdleTimeout   time.Duration

	CircuitBreaker CircuitbreakerOption
	rpcTimeout     time.Duration
	disableCB      bool
}

// WithTimeout config read write timeout
func WithTimeout(timeout time.Duration) Option {
	return Option{func(op *options) {
		op.readWriteTimeout = timeout
		op.rpcTimeout = timeout
	}}
}

// WithConnTimeout config connect timeout, deprecated
func WithConnTimeout(timeout time.Duration) Option {
	return Option{func(op *options) {
		op.connTimeout = timeout
	}}
}

func WithConnMaxRetryTime(d time.Duration) Option {
	return Option{func(op *options) {
		op.connMaxRetryTime = d
	}}
}

func WithLongConnection(maxIdle int, maxIdleTimeout time.Duration) Option {
	return Option{func(op *options) {
		// TODO(zhangyuanjia): 因为server端有默认的3s连接超时, 如果空闲连接大于3s, 可能会从连接池里面取出错误连接使用, 故暂时做此限制;
		_magic_maxIdleTimeout := time.Millisecond * 2500
		if maxIdleTimeout > _magic_maxIdleTimeout {
			maxIdleTimeout = _magic_maxIdleTimeout
			fmt.Fprintf(os.Stderr, "maxIdleTimeout was increased to 2.5s")
		}

		op.useLongPool = true
		op.maxIdle = maxIdle
		op.maxIdleTimeout = maxIdleTimeout
	}}
}

func WithInstances(ins ...*Instance) Option {
	return Option{func(op *options) {
		op.insList = ins
	}}
}

func WithHostPort(hosts ...string) Option {
	return Option{func(op *options) {
		var ins []*Instance
		for _, hostPort := range hosts {
			val := strings.Split(hostPort, ":")
			if len(val) == 2 {
				ins = append(ins, NewInstance(val[0], val[1], nil))
			}
		}
		op.insList = ins
	}}
}

func WithIKService(iks IKService) Option {
	return Option{func(op *options) {
		op.iks = iks
	}}
}

func WithCluster(cluster string) Option {
	return Option{func(op *options) {
		op.cluster = cluster
	}}
}

func WithIDC(idc string) Option {
	return Option{func(op *options) {
		op.idc = idc
	}}
}

func WithWatcher(useWatcher bool) Option {
	return Option{func(op *options) {
		op.useWatcher = useWatcher
	}}
}

func WithCircuitBreaker(errRate float64, minSample int, concurrency int) Option {
	return Option{func(op *options) {
		op.CircuitBreaker = CircuitbreakerOption{
			ErrorRate:      errRate,
			MinSamples:     minSample,
			MaxConcurrency: concurrency,
		}
	}}
}

func WithRPCTimeout(timeout time.Duration) Option {
	return Option{func(op *options) {
		op.rpcTimeout = timeout
	}}
}

func WithDisableCB() Option {
	return Option{func(op *options) {
		op.disableCB = true
	}}
}
