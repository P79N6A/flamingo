/*
   metrics middleware for kite framework
   detail see wiki: https://wiki.bytedance.com/pages/viewpage.action?pageId=51348664
*/
package kitware

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"code.byted.org/kite/endpoint"
	"code.byted.org/kite/kiterrno"
	"code.byted.org/kite/kitutil"
)

type MetricsEmiter interface {
	EmitCounter(name string, value interface{}, prefix string, tagkv map[string]string) error
	EmitTimer(name string, value interface{}, prefix string, tagkv map[string]string) error
	EmitStore(name string, value interface{}, prefix string, tagkv map[string]string) error
}

const NONE string = "none"

// server site metrics format
const (
	successThroughputFmt string = "service.thrift.%s.%s.calledby.success.throughput"
	errorThroughputFmt   string = "service.thrift.%s.%s.calledby.error.throughput"
	successLatencyFmt    string = "service.thrift.%s.%s.calledby.success.latency.us"
	errorLatencyFmt      string = "service.thrift.%s.%s.calledby.error.latency.us"
	accessTotalFmt       string = "service.request.%s.total"

	statusSuccess string = "success"
	statusFailed  string = "failed"
)

// NewAccessMetricsMW create a middleware emit metrics when this EndPoint is called.
func NewAccessMetricsMW(emiter MetricsEmiter) endpoint.Middleware {
	return func(next endpoint.EndPoint) endpoint.EndPoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			if ctx == nil {
				return next(ctx, request)
			}
			begin := time.Now()
			response, err := next(ctx, request)
			took := time.Since(begin).Nanoseconds() / 1000 // us

			// Get servicename from ctx
			sname := kitutil.GetCtxWithDefault(kitutil.GetCtxServiceName, ctx, NONE)

			// Get method name from ctx
			method := kitutil.GetCtxWithDefault(kitutil.GetCtxMethod, ctx, NONE)

			// Get caller from ctx
			caller := kitutil.GetCtxWithDefault(kitutil.GetCtxCaller, ctx, NONE)

			cluster := kitutil.GetCtxWithDefault(kitutil.GetCtxCluster, ctx, "default")
			callerCluster := kitutil.GetCtxWithDefault(kitutil.GetCtxCallerCluster, ctx, "default")

			tname := fmt.Sprintf(successThroughputFmt, sname, method)
			lname := fmt.Sprintf(successLatencyFmt, sname, method)

			if err != nil {
				tname = fmt.Sprintf(errorThroughputFmt, sname, method)
				lname = fmt.Sprintf(errorLatencyFmt, sname, method)
			}

			emiter.EmitCounter(tname, 1, "", map[string]string{
				"from":         caller,
				"from_cluster": callerCluster,
				"to_cluster":   cluster,
			})
			emiter.EmitTimer(lname, took, "", map[string]string{
				"from":         caller,
				"from_cluster": callerCluster,
				"to_cluster":   cluster,
			})

			accessTotal := fmt.Sprintf(accessTotalFmt, sname)
			emiter.EmitCounter(accessTotal, 1, "", map[string]string{
				"from":         caller,
				"from_cluster": callerCluster,
				"to_cluster":   cluster,
			})

			return response, err
		}
	}
}

// client site metrics format
const (
	successRPCThroughputFmt string = "service.thrift.%s.call.success.throughput"
	errorRPCThroughputFmt   string = "service.thrift.%s.call.error.throughput"
	successRPCLatencyFmt    string = "service.thrift.%s.call.success.latency.us"
	errorRPCLatencyFmt      string = "service.thrift.%s.call.error.latency.us"

	stabilityFmt string = "service.stability.%s.throughput"
)

// NewMetricsMW return a middleware for loanrpc calling metric record
func NewRPCMetricsMW(emiter MetricsEmiter) endpoint.Middleware {
	return func(next endpoint.EndPoint) endpoint.EndPoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			if ctx == nil {
				return next(ctx, request)
			}
			begin := time.Now()
			response, err := next(ctx, request)
			took := time.Since(begin).Nanoseconds() / 1000 //us

			code, ok := getRespCode(response)
			if ok && err != nil {
				switch code {
				case kiterrno.NotAllowedByACLCode,
					kiterrno.ForbiddenByDegradationCode:
					return response, err
				}
			}

			sname := kitutil.GetCtxWithDefault(kitutil.GetCtxServiceName, ctx, NONE)
			nsname := kitutil.GetCtxWithDefault(kitutil.GetCtxTargetServiceName, ctx, NONE)
			nmethod := kitutil.GetCtxWithDefault(kitutil.GetCtxTargetMethod, ctx, NONE)

			cluster := kitutil.GetCtxWithDefault(kitutil.GetCtxCluster, ctx, "default")
			targetCluster := kitutil.GetCtxWithDefault(kitutil.GetCtxTargetClusterName, ctx, "default")

			tname := fmt.Sprintf(successRPCThroughputFmt, sname)
			lname := fmt.Sprintf(successRPCLatencyFmt, sname)
			if err != nil {
				tname = fmt.Sprintf(errorRPCThroughputFmt, sname)
				lname = fmt.Sprintf(errorRPCLatencyFmt, sname)
			}

			counterMap := map[string]string{
				"to":           nsname,
				"method":       nmethod,
				"from_cluster": cluster,
				"to_cluster":   targetCluster,
			}
			if err != nil {
				counterMap["err_code"] = strconv.Itoa(code)
			}
			emiter.EmitCounter(tname, 1, "", counterMap)
			emiter.EmitTimer(lname, took, "", map[string]string{
				"to":           nsname,
				"method":       nmethod,
				"from_cluster": cluster,
				"to_cluster":   targetCluster,
			})

			stabilityMetrics := fmt.Sprintf(stabilityFmt, nsname)
			stabilityMap := map[string]string{
				"from":         sname,
				"method":       nmethod,
				"from_cluster": cluster,
				"to_cluster":   targetCluster,
			}
			if code < 0 {
				stabilityMap["label"] = "business_err"
			} else if code >= 100 && code < 200 {
				stabilityMap["label"] = "net_err"
				stabilityMap["err_code"] = strconv.Itoa(code)
			} else {
				stabilityMap["label"] = "success"
			}
			emiter.EmitCounter(stabilityMetrics, 1, "", stabilityMap)

			return response, err
		}
	}
}
