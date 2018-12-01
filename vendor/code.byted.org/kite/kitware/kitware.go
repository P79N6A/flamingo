package kitware

import (
	"context"
	"fmt"
	"log"
	"reflect"

	"code.byted.org/kite/endpoint"
	"code.byted.org/kite/kitutil"
	"code.byted.org/kite/logid"
)

// clientBase implement endpoint.BaseInterface
type clientBase struct {
	logID   string
	caller  string
	client  string
	addr    string
	env     string
	cluster string
}

// GetLogID return logid
func (cb *clientBase) GetLogID() string {
	return cb.logID
}

// GetCaller return caller
func (cb *clientBase) GetCaller() string {
	return cb.caller
}

// GetClient return client
func (cb *clientBase) GetClient() string {
	return cb.client
}

// GetAddr return addr
func (cb *clientBase) GetAddr() string {
	return cb.addr
}

// GetEnv return this request's env
func (cb *clientBase) GetEnv() string {
	return cb.env
}

// GetCluster return upstream's cluster's name
func (cb *clientBase) GetCluster() string {
	return cb.cluster
}

// NewRPCMW return a middleware write the remote servicename and method name into ctx
func NewRPCMW(sname string, mname string) endpoint.Middleware {
	return func(next endpoint.EndPoint) endpoint.EndPoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			ctx = kitutil.NewCtxWithTargetServiceName(ctx, sname)
			ctx = kitutil.NewCtxWithTargetMethod(ctx, mname)
			return next(ctx, request)
		}
	}
}

// NewBaseWriterMW return a middleware write base info into RPC request
func NewBaseWriterMW() endpoint.Middleware {
	return func(next endpoint.EndPoint) endpoint.EndPoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			logID := kitutil.GetCtxWithDefault(kitutil.GetCtxLogID, ctx, "")
			if logID == "" {
				logID = GenLogID(ctx)
			}
			caller := kitutil.GetCtxWithDefault(kitutil.GetCtxServiceName, ctx, "")
			client := kitutil.GetCtxWithDefault(kitutil.GetCtxClient, ctx, "")
			addr := kitutil.GetCtxWithDefault(kitutil.GetCtxLocalIP, ctx, "")
			if addr == "" {
				addr = LocalIP()
			}
			cluster := kitutil.GetCtxWithDefault(kitutil.GetCtxCluster, ctx, "")
			env := kitutil.GetCtxWithDefault(kitutil.GetCtxEnv, ctx, "")
			req, ok := request.(endpoint.KitcCallRequest)
			if !ok {
				log.Printf("request %v not implement KitcCallRequest", request)
				return next(ctx, request)
			}
			req.SetBase(&clientBase{
				logID:   logID,
				caller:  caller,
				client:  client,
				addr:    addr,
				cluster: cluster,
				env:     env,
			})
			return next(ctx, req)
		}
	}
}

type LimitCond interface {
	Limit() int
	Current() int
	Incr()
}

// NewLimitMW return a middleware limit QPS
func NewLimitMW(lc LimitCond) endpoint.Middleware {
	return func(next endpoint.EndPoint) endpoint.EndPoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			if lc.Current() > lc.Limit() {
				return nil, fmt.Errorf("over limit qps, current is %d, limit is %d", lc.Current(), lc.Limit())
			}
			lc.Incr()
			return next(ctx, request)
		}
	}
}

// GenLogID
func GenLogID(ctx context.Context) string {
	return logid.GetNginxID()
}

// NewParserMW return a middleware read from Request.Base and write into ctx,
// this is the first middleware for server processor,
// if logID is none, it will generate a new id and write into ctx.
func NewParserMW() endpoint.Middleware {
	return func(next endpoint.EndPoint) endpoint.EndPoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			// If request not implement standard interface, values in ctx are nil.
			// In this case, should logging something or alarm.
			if r, ok := request.(endpoint.KiteRequest); ok && r.IsSetBase() {
				b := r.GetBase()
				ctx = kitutil.NewCtxWithCaller(ctx, b.GetCaller())
				ctx = kitutil.NewCtxWithAddr(ctx, b.GetAddr())
				ctx = kitutil.NewCtxWithClient(ctx, b.GetClient())
				ctx = kitutil.NewCtxWithEnv(ctx, b.GetEnv())
				ctx = kitutil.NewCtxWithCallerCluster(ctx, b.GetCluster())
				if b.GetLogID() == "" {
					ctx = kitutil.NewCtxWithLogID(ctx, GenLogID(ctx))
				} else {
					ctx = kitutil.NewCtxWithLogID(ctx, b.GetLogID())
				}
			}
			return next(ctx, request)
		}
	}
}

// NewBaseRespCheckMW return a middleware which can check whether the Response's KiteBaseResp is nil.
// If KiteBaseResp is nil, panic this goroutine to generate a warn log.
// This panic will be recovered in kite.RpcServer.processRequests.
func NewBaseRespCheckMW() endpoint.Middleware {
	return func(next endpoint.EndPoint) endpoint.EndPoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			resp, err := next(ctx, request)
			if err != nil {
				return nil, err
			}

			method := kitutil.GetCtxWithDefault(kitutil.GetCtxMethod, ctx, "-")
			sname := kitutil.GetCtxWithDefault(kitutil.GetCtxServiceName, ctx, "-")
			response, ok := resp.(endpoint.KiteResponse)
			if !ok {
				panic(fmt.Sprintf("response type error in %s's %s method. The error type is %s.", sname, method, reflect.TypeOf(resp)))
			}
			if response.GetBaseResp() == nil {
				panic(fmt.Sprintf("response's KiteBaseResp is nil in %s's %s method.", sname, method))
			}
			return response, nil
		}
	}
}
