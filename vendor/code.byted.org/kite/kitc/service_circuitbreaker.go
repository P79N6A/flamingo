package kitc

import (
	"context"
	"errors"
	"strconv"
	"sync"

	"code.byted.org/gopkg/circuitbreaker"
	"code.byted.org/gopkg/logs"
	"code.byted.org/kite/kitutil"
)

const (
	// default open circuitbreaker
	DefaultCircuitBreakerSwitch = 1
	// default one method concurrency loanrpc is 10000
	DefaultMaxConcurrency = 10000
	DefaultErrRate        = 0.5
	DefaultMinSamples     = 200
)

var (
	ErrNoCallTuple     = errors.New("no calltuple information")
	ErrNoTargetService = errors.New("no target service for service circuitbreaker")
	ErrNoTargetMethod  = errors.New("no target method for service breaker")
)

type BreakerMetrics struct {
	From        string  `json:"from"`
	FromCluster string  `json:"from_cluster"`
	To          string  `json:"to"`
	ToCluster   string  `json:"to_cluster"`
	Method      string  `json:"method"`
	State       string  `json:"state"`
	Concurrency int64   `json:"concurrency"`
	Successes   int64   `json:"successes"`
	Failures    int64   `json:"failures"`
	Timeouts    int64   `json:"timeouts"`
	ConseErrors int64   `json:"conse_errors"`
	ErrorRate   float64 `json:"error_rate"`
}

type RichBreaker struct {
	From        string
	FromCluster string
	To          string
	ToCluster   string
	Method      string
	*circuit.Breaker
}

type ServiceCircuit struct {
	errRate     float64
	concurrency int
	minSample   int
	breakers    map[string]*RichBreaker
	l           sync.RWMutex
}

func NewServiceCircuit(errRate float64, concurrency, minSample int) *ServiceCircuit {
	return &ServiceCircuit{
		errRate:     errRate,
		concurrency: concurrency,
		minSample:   minSample,
		breakers:    make(map[string]*RichBreaker),
		l:           sync.RWMutex{},
	}
}

func (sc *ServiceCircuit) Timeout(key string, errRate float64, minSample int) {
	b := sc.findB(key)
	b.TimeoutWithTrip(circuit.RateTripFunc(errRate, int64(minSample)))
}

func (sc *ServiceCircuit) Fail(key string, errRate float64, minSample int) {
	b := sc.findB(key)
	b.FailWithTrip(circuit.RateTripFunc(errRate, int64(minSample)))
}

func (sc *ServiceCircuit) Succeed(key string) {
	b := sc.findB(key)
	b.Succeed()
}

func (sc *ServiceCircuit) Done(key string) {
	b := sc.findB(key)
	b.Done()
}

func (sc *ServiceCircuit) IsAllowed(key string, concurrency int) bool {
	b := sc.findB(key)
	return b.IsAllowed()
}

func (sc *ServiceCircuit) CircuitKey(ctx context.Context) (string, error) {
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

	key := callTuple.To + ":" + callTuple.ToCluster + ":" + callTuple.Method
	sc.record(callTuple.From, callTuple.FromCluster, callTuple.To, callTuple.ToCluster, callTuple.Method, key)
	return key, nil
}

func (sc *ServiceCircuit) AllMetrics() []BreakerMetrics {
	var breakers []*RichBreaker
	sc.l.RLock()
	for _, b := range sc.breakers {
		breakers = append(breakers, b)
	}
	sc.l.RUnlock()
	m := make([]BreakerMetrics, len(breakers))
	for i, b := range breakers {
		m[i].From = b.From
		m[i].FromCluster = b.FromCluster
		m[i].To = b.To
		m[i].ToCluster = b.ToCluster
		m[i].Method = b.Method
		m[i].State = b.State().String()
		m[i].Concurrency = b.Concurrency()
		m[i].Successes = b.Successes()
		m[i].Failures = b.Failures()
		m[i].Timeouts = b.Timeouts()
		m[i].ConseErrors = b.ConseErrors()
		m[i].ErrorRate = b.ErrorRate()
	}
	return m
}

func (sc *ServiceCircuit) findB(key string) *RichBreaker {
	sc.l.RLock()
	b := sc.breakers[key]
	sc.l.RUnlock()
	return b
}

func (sc *ServiceCircuit) record(from, fromCluster, to, toCluster, method string, key string) {
	sc.l.RLock()
	_, ok := sc.breakers[key]
	sc.l.RUnlock()
	if !ok {
		b, _ := circuit.NewBreaker(&circuit.Options{
			ShouldTrip:     circuit.RateTripFunc(sc.errRate, int64(sc.minSample)),
			MaxConcurrency: int64(sc.concurrency),
		})

		richB := &RichBreaker{
			From:        from,
			FromCluster: fromCluster,
			To:          to,
			ToCluster:   toCluster,
			Method:      method,
			Breaker:     b,
		}

		sc.l.Lock()
		if _, ok := sc.breakers[key]; !ok {
			sc.breakers[key] = richB
		}
		sc.l.Unlock()
	}
}

type ServiceCircuitConfig struct {
	storage KVStorage
}

func NewServiceCircuitConfig(storage KVStorage) *ServiceCircuitConfig {
	return &ServiceCircuitConfig{storage: storage}
}

// IsOpen "1", "0"
func (scc *ServiceCircuitConfig) IsOpen(key string) bool {
	val, err := scc.storage.GetOrSet(key, strconv.Itoa(DefaultCircuitBreakerSwitch))
	if err != nil {
		logs.Infof("key=%s, err=%s", key, err)
		return true
	}
	if val == "1" {
		return true
	}
	return false
}

func (scc *ServiceCircuitConfig) MaxConcurrency(key string) int {
	val, err := scc.storage.GetOrSet(key, strconv.Itoa(DefaultMaxConcurrency))
	if err != nil {
		logs.Infof("key=%s, err=%s", key, err)
		return DefaultMaxConcurrency
	}
	num, err := strconv.ParseInt(val, 10, 64)
	if err != nil || num < 0 {
		return DefaultMaxConcurrency
	}
	return int(num)
}

func (scc *ServiceCircuitConfig) ErrRate(key string) float64 {
	val, err := scc.storage.GetOrSet(key, strconv.FormatFloat(DefaultErrRate, 'f', -1, 64))
	if err != nil {
		logs.Infof("key=%s err=%s", key, err)
		return DefaultErrRate
	}
	rate, err := strconv.ParseFloat(val, 64)
	if err != nil || rate <= 0 {
		return DefaultErrRate
	}
	return rate
}

func (scc *ServiceCircuitConfig) MinSample(key string) int {
	val, err := scc.storage.GetOrSet(key, strconv.Itoa(DefaultMinSamples))
	if err != nil {
		logs.Infof("key=%s err=%s", key, err)
		return DefaultMinSamples
	}
	num, err := strconv.ParseInt(val, 10, 64)
	if err != nil || num <= 0 {
		return DefaultMinSamples
	}
	return int(num)
}
