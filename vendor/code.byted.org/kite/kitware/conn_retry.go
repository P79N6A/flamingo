package kitware

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"code.byted.org/gopkg/logs"
	"code.byted.org/kite/endpoint"
	"code.byted.org/kite/kiterrno"
	"code.byted.org/kite/kitutil"
)

// Balancer implements your load balance strategy
type Balancer interface {
	SelectOne() kitutil.Instance // select one instance; if no more instance, just return nil
}

// BalanceRetrier .
type BalanceRetrier interface {
	CreateBalancer(ins []kitutil.Instance) (Balancer, error)
}

// NewConnRetryMW create a connection retry middleware.
// This MW has two main tasks: do load balance strategy and connection retries;
//
// Description:
//   Please see this issue: https://code.byted.org/kite/kitware/issues/3
//
// Context Requires:
//   1. instances
//
// Context Modify:
//   1. put the target instance which should be dialed by downstream into the context
func NewConnRetryMW(br BalanceRetrier) endpoint.Middleware {
	return func(next endpoint.EndPoint) endpoint.EndPoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			ins, ok := kitutil.GetCtxInstances(ctx)
			if !ok || len(ins) == 0 {
				kerr := kiterrno.NewKitErr(kiterrno.NoExpectedFieldCode,
					errors.New("No ins for connecting "))
				return kiterrno.ErrRespNoExpectedField, kerr
			}

			// TODO(xiangchao.01): balancer should contains messages about retry
			banlancer, err := br.CreateBalancer(ins)
			if err != nil {
				kerr := kiterrno.NewKitErr(kiterrno.BadConnBalancerCode, err)
				return kiterrno.ErrRespBadConnBalancer, kerr
			}

			// used to set remote IP for upper MWs
			ripFunc, ripFuncOK := kitutil.GetCtxRIPFunc(ctx)

			var (
				subCtx context.Context
				cancel context.CancelFunc
			)
			deadline, ok := ctx.Deadline()
			if !ok {
				subCtx, cancel = context.WithDeadline(ctx, time.Now().Add(time.Second))
				defer cancel()
			} else {
				subCtx, cancel = context.WithDeadline(ctx, deadline.Add(-2*time.Millisecond))
				defer cancel()
			}
			var resp interface{}
			var errs []error
			targetService, _ := kitutil.GetCtxTargetServiceName(ctx)
			for {
				select {
				case <-subCtx.Done():
					errs = append(errs, errors.New("RPC timeout in conneting Retry"))
					kerr := kiterrno.NewKitErr(kiterrno.ConnRetryCode, joinErrs(errs))
					return kiterrno.ErrRespConnRetry, kerr
				default:
				}

				targetIns := banlancer.SelectOne()
				if targetIns == nil {
					errs = append(errs, errors.New("No more instances to retry"))
					kerr := kiterrno.NewKitErr(kiterrno.ConnRetryCode, joinErrs(errs))
					return kiterrno.ErrRespConnRetry, kerr
				}

				if ripFuncOK {
					ripFunc(targetIns.Host() + ":" + targetIns.Port())
				}

				ctx = kitutil.NewCtxWithTargetInstance(ctx, targetIns)
				resp, err = next(ctx, request)
				if err == nil {
					return resp, err
				}
				errs = append(errs, newConnErr(targetIns, err))

				code, ok := getRespCode(resp)
				if !ok {
					break
				}

				switch code {
				case kiterrno.NotAllowedByInstanceCBCode:
					continue
				case kiterrno.GetConnErrorCode:
					logs.Warnf("get conn for %v err: %v", targetService, err)
					continue
				}
				break
			}

			return resp, err
		}
	}
}

func newConnErr(ins kitutil.Instance, err error) error {
	return fmt.Errorf("ins=%s:%s err=%s", ins.Host(), ins.Port(), err)
}

func joinErrs(errs []error) error {
	s := make([]string, len(errs))
	for i, e := range errs {
		s[i] = e.Error()
	}
	return errors.New(strings.Join(s, ","))
}
