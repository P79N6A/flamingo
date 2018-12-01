package kitware

import (
	"context"
	"errors"
	"net"
	"time"

	"code.byted.org/kite/endpoint"
	"code.byted.org/kite/kiterrno"
	"code.byted.org/kite/kitutil"
)

var (
	// DefaultConnTimeout is 30ms
	DefaultConnTimeout = time.Millisecond * 30
)

// Pooler .
// Pooler has no Put method, which should be implemented by hacking net.Conn's Close method
type Pooler interface {
	Get(ins kitutil.Instance, timeout time.Duration) (net.Conn, error)
}

// NewPoolMW .
// This MW uses the information in the context to create a connection and put it into the context.
//
// Description:
//
// Context Requires:
//   1. connection timeout
//   2. target instance
//
// Context Modify:
//   1. target connection
func NewPoolMW(pooler Pooler) endpoint.Middleware {
	return func(next endpoint.EndPoint) endpoint.EndPoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			instance, ok := kitutil.GetCtxTargetInstance(ctx)
			if ok == false {
				err := errors.New("No Target instance for Pool")
				kerr := kiterrno.NewKitErr(kiterrno.NoExpectedFieldCode, err)
				return kiterrno.ErrRespNoExpectedField, kerr
			}

			// Always use DefaultConnTimeout https://code.byted.org/kite/kitc/issues/26
			conn, err := pooler.Get(instance, DefaultConnTimeout)
			if err != nil {
				kerr := kiterrno.NewKitErr(kiterrno.GetConnErrorCode, err)
				return kiterrno.ErrRespGetConnError, kerr
			}

			ctx = kitutil.NewCtxWithTargetConn(ctx, conn)
			return next(ctx, request)
		}
	}
}
