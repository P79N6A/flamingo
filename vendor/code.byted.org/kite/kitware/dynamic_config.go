package kitware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"code.byted.org/gopkg/logs"
	"code.byted.org/gopkg/net2"
	"code.byted.org/kite/endpoint"
	"code.byted.org/kite/kitutil"
)

type DynamicGetter interface {
	GetByKey(key string) ([]byte, error)
}

// 使用json进行序列化保存
type RPCPolicy struct {
	RetryTimes          int64 // counts of retry
	ConnectTimeout      int64 // ms
	ConnectRetryMaxTime int64 // ms
	ReadTimeout         int64 // ms
	WriteTimeout        int64 // ms
	TrafficPolicy       []kitutil.TPolicy
}

var ErrDyconfigKey = errors.New("DyconfigKey error")

func NewDyconfigKey(ctx context.Context) (string, error) {
	key, err := "", ErrDyconfigKey
	keyItems := make([]string, 8, 8)
	keyItems[0], keyItems[1] = "kite", "config"

	keyItems[2] = getIDCByIP(net2.GetLocalIp())

	caller, ok := kitutil.GetCtxServiceName(ctx)
	if !ok || caller == "" {
		return key, err
	}
	keyItems[3] = caller

	thisCluster, ok := kitutil.GetCtxCluster(ctx)
	if ok && thisCluster != DEFAULT_CLUSTER {
		keyItems[4] = thisCluster
	}

	called, ok := kitutil.GetCtxTargetServiceName(ctx)
	if !ok || called == "" {
		return key, err
	}
	keyItems[5] = called

	targetCluster, ok := kitutil.GetCtxTargetClusterName(ctx)
	if ok && targetCluster != DEFAULT_CLUSTER {
		keyItems[6] = targetCluster
	}

	method, ok := kitutil.GetCtxTargetMethod(ctx)
	if !ok || method == "" {
		return key, err
	}
	keyItems[7] = method

	return EtcdKeyJoin(keyItems), nil
}

type cachedRPCPolicy struct {
	RPCPolicy
	t time.Time
}

func (p *cachedRPCPolicy) Age() time.Duration {
	return time.Since(p.t)
}

var cachedRPCPoliciesMu sync.RWMutex
var cachedRPCPolicies = make(map[string]cachedRPCPolicy)

func getRPCPolicyByKey(dyClient DynamicGetter, key string) (RPCPolicy, error) {
	cachedRPCPoliciesMu.RLock()
	rp, ok := cachedRPCPolicies[key]
	cachedRPCPoliciesMu.RUnlock()
	// TODO(zhangyuanjia): dyClient底部自带cacher, 简称dycacher, 现在模拟出错的情况;
	//  程序启动时, 2个相同的请求打过来, 同时去请求dyClient;
	//  dycacher还未数据, 此时它会阻塞第一个请求, 给第二个请求返回一个emptyErr;
	//  于是dyClient会把默认配置作为第二个请求的配置返回;
	//  第二个请求在此处, 会把这个默认配置, 放入此处的cache;
	//  当第一个请求返回正确配置时, 会发现该key在此处已经有了;
	//  结果是启动后, 到该cache第一次失效时, 动态配置失效;
	// 目前的fix方法为先将此处超时设置短;
	if ok && rp.Age() < 3*time.Second {
		return rp.RPCPolicy, nil
	}
	val, err := dyClient.GetByKey(key)
	if err != nil {
		// TODO(xiangchao.01): always are empty errors
		// logs.Error("Get dynamic config key %s error: %s", keyPath, err)
		return RPCPolicy{}, err
	}
	rpcPolicy := RPCPolicy{}
	err = json.Unmarshal(val, &rpcPolicy)
	cachedRPCPoliciesMu.RLock()
	if rp, ok := cachedRPCPolicies[key]; ok && rp.Age() < 3*time.Second {
		cachedRPCPoliciesMu.RUnlock()
		return rp.RPCPolicy, nil
	}
	cachedRPCPoliciesMu.RUnlock()

	if err != nil {
		return RPCPolicy{}, fmt.Errorf("invalid RPCPolicy value: %v", string(val))
	}

	cachedRPCPoliciesMu.Lock()
	cachedRPCPolicies[key] = cachedRPCPolicy{RPCPolicy: rpcPolicy, t: time.Now()}
	cachedRPCPoliciesMu.Unlock()
	return rpcPolicy, nil
}

// writeConfig read data from etcd and write into ctx
func writeConfig(dyClient DynamicGetter, ctx context.Context, keyPath string) context.Context {
	rpcPolicy, err := getRPCPolicyByKey(dyClient, keyPath)
	if err != nil {
		return ctx
	}
	retryTimes := rpcPolicy.RetryTimes
	ctx = kitutil.NewCtxWithRetryTimes(ctx, int(retryTimes))
	if _, ok := kitutil.GetCtxRPCTimeout(ctx); !ok {
		writeTimeout := time.Duration(time.Duration(rpcPolicy.WriteTimeout) * time.Millisecond)
		if rpcPolicy.WriteTimeout == 0 {
			writeTimeout = 500 * time.Millisecond
		}
		ctx = kitutil.NewCtxWithRPCTimeout(ctx, writeTimeout)
	}
	ctx = kitutil.NewCtxWithTPolicy(ctx, rpcPolicy.TrafficPolicy)
	return ctx
}

func NewDynamicConfigMW(dyClient DynamicGetter) endpoint.Middleware {
	return func(next endpoint.EndPoint) endpoint.EndPoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			key, err := NewDyconfigKey(ctx)
			if err != nil {
				logs.Tracef("key=%s, err=%s dyconfig middleware", key, err)
				return next(ctx, request)
			}
			ctx = writeConfig(dyClient, ctx, key)
			return next(ctx, request)
		}
	}
}

// this function should be removed anytime, so don't expose it
func getIDCByIP(ip string) string {
	hardcodeIDCPrefix := map[string][]string{
		"hy": []string{"10.4"},
		"lf": []string{"10.2", "10.3", "10.6", "10.8", "10.9", "10.11"},
		"va": []string{"10.100"},
		"sg": []string{"10.101"},
	}
	ip = strings.TrimSpace(ip)
	for idc, prefixes := range hardcodeIDCPrefix {
		for _, prefix := range prefixes {
			if strings.HasPrefix(ip, prefix) {
				return idc
			}
		}
	}
	return "default"
}
