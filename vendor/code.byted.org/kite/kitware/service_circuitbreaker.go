package kitware

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"

	"context"

	"code.byted.org/kite/endpoint"
	"code.byted.org/kite/kiterrno"
	"code.byted.org/kite/kitutil"
)

// Circuitbreaker should be implemented in command pattern
type ServiceCircuitbreaker interface {
	CircuitKey(ctx context.Context) (string, error)
	IsAllowed(key string, concurrency int) bool
	Timeout(key string, errRate float64, minSample int)
	Fail(key string, errRate float64, minSample int)
	Succeed(key string)
	Done(key string)
}

// NewServiceBreakerMW creates a circuitbreaker middleware to control this RPC in service granularity;
// This MW should be in front of all other MWs so that it can control the whole RPC timeout.
//
// Description:
//
// Context Requires:
//   1. RPC timeout
//
// Context Modify:
//   nothing
func NewServiceBreakerMW(breaker ServiceCircuitbreaker) endpoint.Middleware {
	return func(next endpoint.EndPoint) endpoint.EndPoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			cbKey, err := breaker.CircuitKey(ctx)
			if err != nil {
				kerr := kiterrno.NewKitErr(kiterrno.NoExpectedFieldCode, err)
				return kiterrno.ErrRespNoExpectedField, kerr
			}

			rpcTimeout, ok := kitutil.GetCtxRPCTimeout(ctx)
			if !ok {
				kerr := kiterrno.NewKitErr(kiterrno.NoExpectedFieldCode,
					errors.New("No RPC Timeout for Service Breaker"))
				return kiterrno.ErrRespNoExpectedField, kerr
			}
			if rpcTimeout <= 0 {
				kerr := kiterrno.NewKitErr(kiterrno.NoExpectedFieldCode,
					fmt.Errorf("Invalid RPC Timeout: %v", rpcTimeout))
				return kiterrno.ErrRespNoExpectedField, kerr
			}

			cnf, _ := kitutil.GetCtxCBConfig(ctx)

			if !breaker.IsAllowed(cbKey, cnf.MaxConcurrency) {
				targetService, _ := kitutil.GetCtxTargetServiceName(ctx)
				targetMethod, _ := kitutil.GetCtxTargetMethod(ctx)
				err := fmt.Errorf("service=%s method=%s", targetService, targetMethod)
				kerr := kiterrno.NewKitErr(kiterrno.NotAllowedByServiceCBCode, err)
				return kiterrno.ErrRespNotAllowedByServiceCB, kerr
			}
			_, ok = ctx.Deadline()
			if !ok {
				subCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
				defer cancel()
				ctx = subCtx
			}
			done := make(chan struct{}, 1)
			var resp interface{}

			// used by conn_retry MW to set remote IP
			var rip string
			var ripLock sync.Mutex
			ripFunc := func(remoteIP string) {
				ripLock.Lock()
				rip = remoteIP
				ripLock.Unlock()
			}
			ctx = kitutil.NewCtxWithRIPFunc(ctx, ripFunc)

			// Async RPC to control timeout precisely
			go func() {
				defer func() {
					done <- struct{}{}
					breaker.Done(cbKey)
					if e := recover(); e != nil {
						const size = 64 << 10
						buf := make([]byte, size)
						buf = buf[:runtime.Stack(buf, false)]
						// write into stderr for collecting
						fmt.Fprintf(os.Stderr, "%s\n%s\n", e, buf)
					}
				}()
				resp, err = next(ctx, request)
			}()

			useBreaker, _ := kitutil.GetCtxBreakerSwitch(ctx)
			select {
			// Not timeout, but may failed
			case <-done:
				// return directly
				if !useBreaker {
					breaker.Succeed(cbKey)
					return resp, err
				}

				if err == nil { // succeed
					breaker.Succeed(cbKey)
					return resp, err
				}

				// failed and using error code to decide if ignore it
				// if response is nil, regard it as success, nil response will be dealed with another middleware
				code, ok := getRespCode(resp)
				if !ok {
					// ignore
					return resp, err
				}
				switch code {
				// ignore all internal errors(like NoExpectedField, IDCSelectError) and
				// all ACL and degradation errors, and
				// all RPC timeout errors which have already been recored when the MW receive this error;
				case kiterrno.NotAllowedByACLCode,
					kiterrno.ForbiddenByDegradationCode,
					kiterrno.GetDegradationPercentErrorCode,
					kiterrno.RPCTimeoutCode,
					kiterrno.BadConnBalancerCode,
					kiterrno.BadConnRetrierCode,
					kiterrno.BadRPCRetrierCode,
					kiterrno.NoExpectedFieldCode,
					kiterrno.ServiceDiscoverCode,
					kiterrno.IDCSelectErrorCode:
				// regard all network errors and relative errors caused by network as failed
				case kiterrno.NotAllowedByServiceCBCode,
					kiterrno.NotAllowedByInstanceCBCode,
					kiterrno.ConnRetryCode,
					kiterrno.GetConnErrorCode,
					kiterrno.ReadTimeoutCode,
					kiterrno.WriteTimeoutCode,
					kiterrno.ConnResetByPeerCode:
					breaker.Fail(cbKey, cnf.ErrRate, cnf.MinSample)
				default:
					breaker.Succeed(cbKey)
				}
				return resp, err
			case <-ctx.Done(): // RPC timeout
				if useBreaker {
					breaker.Timeout(cbKey, cnf.ErrRate, cnf.MinSample)
				}
				kerr := kiterrno.NewKitErr(kiterrno.RPCTimeoutCode, ctx.Err())

				ripLock.Lock()
				curRIP := rip
				ripLock.Unlock()

				return kiterrno.NewErrRespWithAddr(kiterrno.RPCTimeoutCode, curRIP), kerr
			}
		}
	}
}
