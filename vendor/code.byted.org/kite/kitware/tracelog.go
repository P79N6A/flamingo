package kitware

import (
	"context"
	"strconv"
	"time"

	"code.byted.org/kite/endpoint"
	"code.byted.org/kite/kiterrno"
	"code.byted.org/kite/kitutil"
)

type TraceLogger interface {
	Trace(format string, v ...interface{})
	Error(format string, v ...interface{})
}

// NewAccessLogMW used for logging access log in a service' server side.
func NewAccessLogMW(logger TraceLogger) endpoint.Middleware {
	return func(next endpoint.EndPoint) endpoint.EndPoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			code := int32(kiterrno.SuccessCode)
			begin := time.Now()
			response, err := next(ctx, request)
			if err != nil {
				code = int32(kiterrno.UserErrorCode)
			}
			if resp, ok := response.(endpoint.KiteResponse); ok {
				if bp := resp.GetBaseResp(); bp != nil {
					code = bp.GetStatusCode()
				}
			}
			cost := time.Since(begin).Nanoseconds() / 1000 //us
			// access log
			localIP := kitutil.GetCtxWithDefault(kitutil.GetCtxLocalIP, ctx, "-")
			sname := kitutil.GetCtxWithDefault(kitutil.GetCtxServiceName, ctx, "-")
			cluster := kitutil.GetCtxWithDefault(kitutil.GetCtxCluster, ctx, "-")
			logID := kitutil.GetCtxWithDefault(kitutil.GetCtxLogID, ctx, "-")
			method := kitutil.GetCtxWithDefault(kitutil.GetCtxMethod, ctx, "-")
			addr := kitutil.GetCtxWithDefault(kitutil.GetCtxAddr, ctx, "-")
			caller := kitutil.GetCtxWithDefault(kitutil.GetCtxCaller, ctx, "-")
			callerCluster := kitutil.GetCtxWithDefault(kitutil.GetCtxCallerCluster, ctx, "-")
			env := kitutil.GetCtxWithDefault(kitutil.GetCtxEnv, ctx, "-")
			ss := formatLog(localIP, sname, logID, cluster, method, addr, caller, callerCluster, cost, int64(code), env)
			if err != nil {
				logger.Error("%s", ss)
			} else {
				logger.Trace("%s", ss)
			}
			return response, err
		}
	}
}

func formatLog(ip, psm, logid, cluster, method, rip, rname, rcluster string, cost, code int64, env string) string {
	b := make([]byte, 0, 4096)
	b = append(b, ip...)
	b = append(b, ' ')
	b = append(b, psm...)
	b = append(b, ' ')
	b = append(b, logid...)
	b = append(b, ' ')
	b = append(b, cluster...)
	b = append(b, " method="...)
	b = append(b, method...)
	b = append(b, " rip="...)
	b = append(b, rip...)
	b = append(b, " called="...)
	b = append(b, rname...)
	b = append(b, " cluster="...)
	b = append(b, rcluster...)
	b = append(b, " cost="...)
	b = strconv.AppendInt(b, cost, 10)
	b = append(b, " status="...)
	b = strconv.AppendInt(b, code, 10)
	b = append(b, " env="...)
	b = append(b, env...)
	return string(b)
}

// NewRPCLogMW return a middleware for logging RPC record
func NewRPCLogMW(logger TraceLogger) endpoint.Middleware {
	return func(next endpoint.EndPoint) endpoint.EndPoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			code := int32(kiterrno.SuccessCode)
			begin := time.Now()
			response, err := next(ctx, request)
			if err != nil {
				code = int32(kiterrno.UserErrorCode)
			}
			rip := "-"
			if resp, ok := response.(endpoint.KitcCallResponse); ok {
				if bp := resp.GetBaseResp(); bp != nil {
					code = bp.GetStatusCode()
				}
				rip = resp.RemoteAddr()
			}
			cost := time.Since(begin).Nanoseconds() / 1000 // us

			tp, ok := kitutil.GetCtxCallTuple(ctx)
			if !ok {
				tp = &kitutil.CallTuple{}
				tp.From = "-"
				tp.FromCluster = "-"
				tp.To = "-"
				tp.ToCluster = "-"
				tp.Method = "-"
			}

			defaultString := func(s, d string) string {
				if s == "" {
					return d
				}
				return s
			}
			logID := kitutil.GetCtxWithDefault(kitutil.GetCtxLogID, ctx, "-")
			localIP := LocalIP()
			caller := defaultString(tp.From, "-")
			cluster := defaultString(tp.FromCluster, "-")
			method := defaultString(tp.Method, "-")
			rname := defaultString(tp.To, "-")
			targetCluster := defaultString(tp.ToCluster, "-")
			env := kitutil.GetCtxWithDefault(kitutil.GetCtxEnv, ctx, "-")
			ss := formatLog(localIP, caller, logID, cluster, method, rip, rname, targetCluster, cost, int64(code), env)
			if err != nil {
				logger.Error("%s", ss)
			} else {
				logger.Trace("%s", ss)
			}
			return response, err
		}
	}
}
