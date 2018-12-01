package kitware

import (
	"context"

	"code.byted.org/kite/endpoint"
)

// NewConfigReporterMW .
//
// Description:
//	this MW used to report config in context on runtime;
//
// Context Requires:
//
// Context Modify:
func NewConfigReporterMW(report func(ctx context.Context)) endpoint.Middleware {
	safeReport := func(ctx context.Context) {
		defer func() {
			recover()
		}()
		report(ctx)
	}

	return func(next endpoint.EndPoint) endpoint.EndPoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			safeReport(ctx)
			return next(ctx, request)
		}
	}
}
