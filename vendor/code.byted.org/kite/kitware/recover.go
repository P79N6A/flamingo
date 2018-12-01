package kitware

import (
	"context"
	"runtime"

	"code.byted.org/gopkg/logs"
	"code.byted.org/kite/endpoint"
)

const RecoverMW = "RecoverMW"

func NewRecoverMW() endpoint.Middleware {
	return func(next endpoint.EndPoint) endpoint.EndPoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			defer func() {
				if e := recover(); e != nil {
					const size = 64 << 10
					buf := make([]byte, size)
					buf = buf[:runtime.Stack(buf, false)]
					logs.CtxError(ctx, "Panic: Request is: %v", request)
					logs.CtxError(ctx, "KITE: panic in handler: %s: %s", e, buf)
					panic(RecoverMW)
				}
			}()
			return next(ctx, request)
		}
	}
}
