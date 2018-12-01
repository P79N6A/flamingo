package kitware

import (
	"context"
	"fmt"

	"code.byted.org/kite/endpoint"
	"code.byted.org/kite/kiterrno"
	"code.byted.org/kite/kitutil"
)

// Discoverer 服务发现接口
// 服务名应该在其创建的时候被指定
type Discoverer interface {
	Name() string // 返回目标服务名
	Lookup(idc, cluster, env string) ([]kitutil.Instance, error)
}

// NewServiceDiscoverMW create a service discover middleware.
// In this middleware, read idc from context, then write service's instances into
// context.
//
// Description:
//   ...
//
// Context Requires:
//   1. idc
//
// Context Modify:
//   1. list of remote service's instancs
func NewServiceDiscoverMW(discoverer Discoverer) endpoint.Middleware {
	return func(next endpoint.EndPoint) endpoint.EndPoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			idc, ok := kitutil.GetCtxIDC(ctx)
			if !ok || idc == "" {
				return kiterrno.ErrRespNoExpectedField, fmt.Errorf("No IDC")
			}
			// targetCluster and env can be empty
			targetCluster, _ := kitutil.GetCtxTargetClusterName(ctx)
			env, _ := kitutil.GetCtxEnv(ctx)

			ins, err := discoverer.Lookup(idc, targetCluster, env)
			if err != nil {
				kerr := kiterrno.NewKitErr(kiterrno.ServiceDiscoverCode, err)
				return kiterrno.ErrRespServiceDiscover, kerr
			}
			if len(ins) == 0 {
				kerr := kiterrno.NewKitErr(kiterrno.ServiceDiscoverCode,
					fmt.Errorf("No service discovered idc=%s service=%s cluster=%s env=%s",
						idc, discoverer.Name(), targetCluster, env))
				return kiterrno.ErrRespServiceDiscover, kerr
			}

			ctx = kitutil.NewCtxWithInstances(ctx, ins)
			return next(ctx, request)
		}
	}
}
