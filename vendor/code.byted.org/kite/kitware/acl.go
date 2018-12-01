package kitware

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"code.byted.org/kite/endpoint"
	"code.byted.org/kite/kiterrno"
	"code.byted.org/kite/kitutil"
)

const (
	ACL_ALLOW     = "0"
	ACL_NOT_ALLOW = "1"

	DEFAULT_CLUSTER = "default"
)

type ACLer interface {
	GetByKey(key string) (string, error)
}

// NewRPCAclKey
// /kite/acl/from/{from_cluster}/to/{to_cluster}/method
func NewRPCAclKey(ctx context.Context) (string, error) {

	keyItems := make([]string, 7, 7)
	keyItems[0], keyItems[1] = "kite", "acl"

	// upstream service name
	caller, ok := kitutil.GetCtxServiceName(ctx)
	if !ok || caller == "" {
		return "", errNoServiceName
	}
	keyItems[2] = caller

	// upstream service cluster name
	thisCluster, ok := kitutil.GetCtxCluster(ctx)
	if ok && thisCluster != DEFAULT_CLUSTER {
		keyItems[3] = thisCluster
	}

	// target service name
	sname, ok := kitutil.GetCtxTargetServiceName(ctx)
	if !ok || sname == "" {
		return "", errNoTargetServiceName
	}
	keyItems[4] = sname

	// target service cluster name
	targetCluster, ok := kitutil.GetCtxTargetClusterName(ctx)
	if ok && targetCluster != DEFAULT_CLUSTER {
		keyItems[5] = targetCluster
	}

	// target service method
	mname, ok := kitutil.GetCtxTargetMethod(ctx)
	if !ok || mname == "" {
		return "", errNoTargetServiceMethod
	}
	keyItems[6] = mname

	return EtcdKeyJoin(keyItems), nil
}

func NewRPCACLMW(acler ACLer) endpoint.Middleware {
	return func(next endpoint.EndPoint) endpoint.EndPoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			key, err := NewRPCAclKey(ctx)
			if err != nil {
				return next(ctx, request)
			}
			val, err := acler.GetByKey(key)
			if err != nil {
				return next(ctx, request)
			}
			val = strings.TrimSpace(val)
			if val == ACL_NOT_ALLOW {
				kErr := kiterrno.NewKitErr(kiterrno.NotAllowedByACLCode, nil)
				return kiterrno.ErrRespNotAllowedByACL, kErr
			}
			return next(ctx, request)
		}
	}
}

// NewAccessAclKey
// /kite/acl/from/{from_cluster}/to/{to_cluster}/method
func NewAccessAclKey(ctx context.Context) (string, error) {
	keyItems := make([]string, 7, 7)
	keyItems[0], keyItems[1] = "kite", "acl"

	// upstream service name
	caller, ok := kitutil.GetCtxCaller(ctx)
	if !ok || caller == "" {
		// if there is no caller use "none"
		caller = NONE
	}
	keyItems[2] = caller

	// upstream service cluster name
	callerCluster, ok := kitutil.GetCtxCallerCluster(ctx)
	if ok && callerCluster != DEFAULT_CLUSTER {
		keyItems[3] = callerCluster
	}

	// target service name
	sname, ok := kitutil.GetCtxServiceName(ctx)
	if !ok || sname == "" {
		return "", errors.New("no service name")
	}
	keyItems[4] = sname

	// target service cluster name
	thisCluster, ok := kitutil.GetCtxCluster(ctx)
	if ok && thisCluster != DEFAULT_CLUSTER {
		keyItems[5] = thisCluster
	}

	// target service method
	mname, ok := kitutil.GetCtxMethod(ctx)
	if !ok || mname == "" {
		return "", errors.New("no service's  method")
	}
	keyItems[6] = mname

	return EtcdKeyJoin(keyItems), nil
}

func NewAccessACLMW(acler ACLer) endpoint.Middleware {
	return func(next endpoint.EndPoint) endpoint.EndPoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			key, err := NewAccessAclKey(ctx)
			if err != nil {
				return next(ctx, request)
			}
			val, err := acler.GetByKey(key)
			if err != nil {
				return next(ctx, request)
			}
			val = strings.TrimSpace(val)
			if val == ACL_NOT_ALLOW {
				caller := kitutil.GetCtxWithDefault(kitutil.GetCtxCaller, ctx, NONE)
				kErr := kiterrno.NewKitErr(kiterrno.NotAllowedByACLCode,
					fmt.Errorf("caller=%s", caller))
				return nil, kErr
			}
			return next(ctx, request)
		}
	}
}
