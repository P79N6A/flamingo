package kite

import (
	"context"

	"code.byted.org/kite/endpoint"
	"code.byted.org/kite/kitc"
	"code.byted.org/kite/kitutil"
	"code.byted.org/kite/kitware"
)

// MethodContext return a context with method name key
func MethodContext(method string) context.Context {
	ctx := kitutil.NewCtxWithServiceName(context.Background(), ServiceName)
	ctx = kitutil.NewCtxWithLocalIP(ctx, LocalIp)
	ctx = kitutil.NewCtxWithMethod(ctx, method)
	ctx = kitutil.NewCtxWithCluster(ctx, ServiceCluster)
	return ctx
}

// KiteMiddle wrap every endpoint in this service.
func KiteMW(next endpoint.EndPoint) endpoint.EndPoint {
	mids := []endpoint.Middleware{
		kitware.NewAccessLogMW(AccessLogger),
		kitware.NewAccessMetricsMW(metricsClient),
		kitware.NewAccessACLMW(kitc.NewAcler(etcdCache)),
		kitware.NewRecoverMW(),
		NewAdditionMW(),
		kitware.NewBaseRespCheckMW(),
	}
	mid := endpoint.Chain(kitware.NewParserMW(), mids...)
	return mid(next)
}

var mMap = make(map[string]endpoint.Middleware)
var userMW endpoint.Middleware

// AddMethodMW use a middleware for a define method.
func AddMethodMW(m string, mws ...endpoint.Middleware) {
	if len(mws) >= 1 {
		mMap[m] = endpoint.Chain(mws[0], mws[1:]...)
	}
}

// Use middlewares will enable for all this service's method.
func Use(mws ...endpoint.Middleware) {
	if len(mws) >= 1 {
		userMW = endpoint.Chain(mws[0], mws[1:]...)
	}
}

func NewAdditionMW() endpoint.Middleware {
	return func(next endpoint.EndPoint) endpoint.EndPoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			method, ok := kitutil.GetCtxMethod(ctx)
			if !ok {
				method = "-"
			}
			if mw, ok := mMap[method]; ok {
				next = mw(next)
			}
			if userMW != nil {
				next = userMW(next)
			}
			return next(ctx, request)
		}
	}
}
