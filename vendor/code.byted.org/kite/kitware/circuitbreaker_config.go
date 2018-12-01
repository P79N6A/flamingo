package kitware

import (
	"context"
	"errors"

	"code.byted.org/kite/endpoint"
	"code.byted.org/kite/kitutil"
)

var (
	errNoCallTuple           = errors.New("no calltuple in context")
	errNoServiceName         = errors.New("no service name")
	errNoTargetServiceName   = errors.New("no target service name")
	errNoTargetServiceMethod = errors.New("no target service name method")
)

type ServiceCircuitBreakerConfiger interface {
	IsOpen(key string) bool
	MaxConcurrency(key string) int
	ErrRate(key string) float64
	MinSample(key string) int
}

// FromServiceCircuitBreakerKey ...
// /kite/circuitbreaker/switch/from/from_cluster/to/to_cluster/method
func FromServiceCircuitBreakerSwitchPath(ctx context.Context) (string, error) {
	callTuple, ok := kitutil.GetCtxCallTuple(ctx)
	if !ok {
		return "", errNoCallTuple
	}
	key := CallTupleKey{Prefix: "kite/circuitbreaker/switch", CallTuple: *callTuple}
	return EtcdCallTuplePropKey(key), nil
}

// NewServiceCircuitBreakerMaxConcurrentKey
// /kite/circuitbreaker/config/from/from_cluster/to/to_cluster/method/concurrency
func FromServiceCircuitBreakerMaxConcurrencyPath(ctx context.Context) (string, error) {
	callTuple, ok := kitutil.GetCtxCallTuple(ctx)
	if !ok {
		return "", errNoCallTuple
	}
	key := CallTupleKey{Prefix: "kite/circuitbreaker/config", CallTuple: *callTuple, Suffix: "concurrency"}
	return EtcdCallTuplePropKey(key), nil

}

// /kite/circuitbreaker/config/from/from_cluster/to/to_cluster/method/errRate
func FromServiceCircuitBreakerErrRatePath(ctx context.Context) (string, error) {
	callTuple, ok := kitutil.GetCtxCallTuple(ctx)
	if !ok {
		return "", errNoCallTuple
	}
	key := CallTupleKey{Prefix: "kite/circuitbreaker/config", CallTuple: *callTuple, Suffix: "errRate"}
	return EtcdCallTuplePropKey(key), nil

}

// /kite/circuitbreaker/config/from/from_cluster/to/to_cluster/method/minSample
func FromServiceCircuitBreakerMinSamplePath(ctx context.Context) (string, error) {
	callTuple, ok := kitutil.GetCtxCallTuple(ctx)
	if !ok {
		return "", errNoCallTuple
	}
	key := CallTupleKey{Prefix: "kite/circuitbreaker/config", CallTuple: *callTuple, Suffix: "minSample"}
	return EtcdCallTuplePropKey(key), nil
}

func NewServiceCircuitBreakerConfigMW(configer ServiceCircuitBreakerConfiger) endpoint.Middleware {
	return func(next endpoint.EndPoint) endpoint.EndPoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			switchKey, _ := FromServiceCircuitBreakerSwitchPath(ctx)
			state := configer.IsOpen(switchKey)
			if _, ok := kitutil.GetCtxBreakerSwitch(ctx); !ok {
				ctx = kitutil.NewCtxWithBreakerSwitch(ctx, state)
			}
			if _, ok := kitutil.GetCtxCBConfig(ctx); ok {
				return next(ctx, request)
			}
			concurrentKey, _ := FromServiceCircuitBreakerMaxConcurrencyPath(ctx)
			num := configer.MaxConcurrency(concurrentKey)
			errRateKey, _ := FromServiceCircuitBreakerErrRatePath(ctx)
			rate := configer.ErrRate(errRateKey)
			minSampleKey, _ := FromServiceCircuitBreakerMinSamplePath(ctx)
			minSample := configer.MinSample(minSampleKey)
			ctx = kitutil.NewCtxWithCBConfig(ctx, &kitutil.CBConfig{
				MaxConcurrency: num,
				ErrRate:        rate,
				MinSample:      minSample,
			})
			return next(ctx, request)
		}
	}
}
