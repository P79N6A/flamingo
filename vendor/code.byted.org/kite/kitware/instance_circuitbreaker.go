package kitware

import (
	"context"
	"fmt"

	"code.byted.org/kite/endpoint"
	"code.byted.org/kite/kiterrno"
	"code.byted.org/kite/kitutil"
)

// Circuitbreaker should be implemented in command pattern
type InstanceCircuitbreaker interface {
	CircuitKey(ctx context.Context) (string, error)
	IsAllowed(key string) bool
	Timeout(key string)
	Fail(key string)
	Succeed(key string)
	Done(key string)
}

// NewInstanceBreakerMW create a circuitbreaker middleware in instance level.
// This MW implements circuitbreaker pattern for kite.
//
// Description:
//
// Context Requires:
//   1. taget instance
//
// Context Modify:
//   nothing
func NewInstanceBreakerMW(breaker InstanceCircuitbreaker) endpoint.Middleware {
	return func(next endpoint.EndPoint) endpoint.EndPoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			key, err := breaker.CircuitKey(ctx)
			if err != nil {
				kerr := kiterrno.NewKitErr(kiterrno.NoExpectedFieldCode, err)
				return kiterrno.ErrRespNoExpectedField, kerr
			}

			if !breaker.IsAllowed(key) {
				targetService, _ := kitutil.GetCtxTargetServiceName(ctx)
				targetMethod, _ := kitutil.GetCtxTargetMethod(ctx)
				ins, _ := kitutil.GetCtxTargetInstance(ctx)
				err := fmt.Errorf("service=%s method=%s hostport=%s",
					targetService, targetMethod, key)
				kerr := kiterrno.NewKitErr(kiterrno.NotAllowedByInstanceCBCode, err)
				return kiterrno.NewErrRespWithAddr(kiterrno.NotAllowedByInstanceCBCode, ins.Host()+":"+ins.Port()), kerr
			}
			defer breaker.Done(key)

			resp, err := next(ctx, request)
			if err == nil {
				breaker.Succeed(key)
				return resp, err
			}

			// failed and using error code to decide if ignore it
			// if response is nil, regard it as success, because this RPC is done.
			code, ok := getRespCode(resp)
			if !ok { // ignore
				return resp, err
			}

			// only control connection error
			switch code {
			case kiterrno.GetConnErrorCode:
				breaker.Fail(key)
			default:
				breaker.Succeed(key)
			}
			return resp, err
		}
	}
}
