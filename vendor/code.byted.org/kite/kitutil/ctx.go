package kitutil

import (
	"context"
	"net"
	"time"
)

const (
	LOGIDKEY         = "K_LOGID"         // 唯一的Request ID
	CALLERKEY        = "K_CALLER"        // 上游服务的名字
	CALLERCLUSTERKEY = "K_CALLERCLUSTER" // 上游服务的集群名字
	ENVKEY           = "K_ENV"           // 上游服务带过来的环境参数
	CLIENTKEY        = "K_CLIENT"        // 客户端的标识，目前保留
	ADDRKEY          = "K_ADDR"          // 上游服务的IP地址
	LOCALIPKEY       = "K_LOCALIP"       // 本服务的IP 地址
	METHODKEY        = "K_METHOD"        // 本服务当前所处的接口名字（也就是Method名字）
	SNAMEKEY         = "K_SNAME"         // 本服务的名字
	CLUSTERKEY       = "K_CLUSTER"       // 本服务集群的名字

	NSNAMEKEY   = "K_NSNAME"   // 下游服务名字
	NCLUSTERKEY = "K_NCLUSTER" // 下游服务集群的名字
	NMNAMEKEY   = "K_NMNAME"   // 下游服务的接口名字
	NIDCKEY     = "K_IDC"      // 下游服务选择的IDC名字

	CONNECTTIMEOUTKEY      = "K_CONN_TIMEOUT"        // RPC调用连接超时
	CONNECTRETRYMAXTIMEKEY = "K_CONN_RETRY_MAX_TIME" // RPC调用重试最长时间
	READTIMEOUTKEY         = "K_READ_TIMEOUT"        // RPC调用连接的读超时
	WRITETIMEOUTKEY        = "K_WRITE_TIMEOUT"       // RPC调用连接的写超时
	TRAFFICPOLICYKEY       = "K_TRAFFIC_POLICY"      // RPC调用的机房流量策略
	RETRYTIMES             = "K_RETRY_TIMES"         // RPC调用业务层的重试次数
	INSTANCESKEY           = "K_INSTANCES"           // RPC调用的机器实例

	TARGET_INSTANCE_KEY     = "K_TARGET_INSTANCE"     // target loanrpc instance to be called
	DEGRADATION_PERCENT_KEY = "K_DEGRADATION_PERCENT" // degradation percent
	TARGET_CONNECTION_KEY   = "K_TARGET_CONNECTION"   //
	RPC_TIMEOUT_KEY         = "K_RPC_TIMEOUT"         // total timeout for this RPC
	AUTO_RPC_RETRY_KEY      = "K_AUTO_RPC_RETRY"      //
	BREAKER_SWITCH_KEY      = "K_BREAKER_SWITCH"      // if use circuitbreaker
)

type (
	CallTupleKey   struct{}
	CBConfigKey    struct{}
	RetryConfigKey struct{}
	ConstIDCKey    struct{}
	RIPFuncKey     struct{}
)

// getStrCtx read the value of key in ctx, return it in string type.
func getStrCtx(ctx context.Context, key string) (string, bool) {
	if ctx == nil {
		return "", false
	}
	v := ctx.Value(key)
	switch v := v.(type) {
	case string:
		return v, true
	case *string:
		return *v, true
	}
	return "", false
}

// GetCtxIDC x.
func GetCtxIDC(ctx context.Context) (string, bool) {
	return getStrCtx(ctx, NIDCKEY)
}

// NewCtxWithIDC x
func NewCtxWithIDC(ctx context.Context, idc string) context.Context {
	return context.WithValue(ctx, NIDCKEY, idc)
}

// GetCtxLogID return logid store in ctx.
func GetCtxLogID(ctx context.Context) (string, bool) {
	return getStrCtx(ctx, LOGIDKEY)
}

// NewCtxWithLogID
func NewCtxWithLogID(ctx context.Context, logID string) context.Context {
	return context.WithValue(ctx, LOGIDKEY, logID)
}

// GetCtxCaller
func GetCtxCaller(ctx context.Context) (string, bool) {
	return getStrCtx(ctx, CALLERKEY)
}

// NewCtxWithCaller
func NewCtxWithCaller(ctx context.Context, caller string) context.Context {
	return context.WithValue(ctx, CALLERKEY, caller)
}

// GetCtxEnv
func GetCtxEnv(ctx context.Context) (string, bool) {
	return getStrCtx(ctx, ENVKEY)
}

// NewCtxWithEnv
func NewCtxWithEnv(ctx context.Context, env string) context.Context {
	return context.WithValue(ctx, ENVKEY, env)
}

// GetCtxCluster
func GetCtxCluster(ctx context.Context) (string, bool) {
	return getStrCtx(ctx, CLUSTERKEY)
}

// NewCtxWithCluster
func NewCtxWithCluster(ctx context.Context, cluster string) context.Context {
	return context.WithValue(ctx, CLUSTERKEY, cluster)
}

// GetCtxCallerCluster
func GetCtxCallerCluster(ctx context.Context) (string, bool) {
	return getStrCtx(ctx, CALLERCLUSTERKEY)
}

// NewCtxWithCallerCluster
func NewCtxWithCallerCluster(ctx context.Context, cluster string) context.Context {
	return context.WithValue(ctx, CALLERCLUSTERKEY, cluster)
}

// GetCtxClient
func GetCtxClient(ctx context.Context) (string, bool) {
	return getStrCtx(ctx, CLIENTKEY)
}

// NewCtxWithClient
func NewCtxWithClient(ctx context.Context, client string) context.Context {
	return context.WithValue(ctx, CLIENTKEY, client)
}

// GetCtxAddr
func GetCtxAddr(ctx context.Context) (string, bool) {
	return getStrCtx(ctx, ADDRKEY)
}

// NewCtxWithAddr
func NewCtxWithAddr(ctx context.Context, addr string) context.Context {
	return context.WithValue(ctx, ADDRKEY, addr)
}

// GetCtxLocalIP
func GetCtxLocalIP(ctx context.Context) (string, bool) {
	return getStrCtx(ctx, LOCALIPKEY)
}

// NewCtxWithLocalIP
func NewCtxWithLocalIP(ctx context.Context, localIP string) context.Context {
	return context.WithValue(ctx, LOCALIPKEY, localIP)
}

// GetCtxMethod
func GetCtxMethod(ctx context.Context) (string, bool) {
	return getStrCtx(ctx, METHODKEY)
}

// NewCtxWithMethod
func NewCtxWithMethod(ctx context.Context, method string) context.Context {
	return context.WithValue(ctx, METHODKEY, method)
}

// GetCtxServiceName
func GetCtxServiceName(ctx context.Context) (string, bool) {
	return getStrCtx(ctx, SNAMEKEY)
}

// NewCtxWithServiceName
func NewCtxWithServiceName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, SNAMEKEY, name)
}

// GetCtxTargetServiceName
func GetCtxTargetServiceName(ctx context.Context) (string, bool) {
	return getStrCtx(ctx, NSNAMEKEY)
}

// NewCtxWithTargetServiceName
func NewCtxWithTargetServiceName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, NSNAMEKEY, name)
}

// GetCtxTargetClusterName
func GetCtxTargetClusterName(ctx context.Context) (string, bool) {
	return getStrCtx(ctx, NCLUSTERKEY)
}

// NewCtxWithTargetCluster
func NewCtxWithTargetClusterName(ctx context.Context, cluster string) context.Context {
	return context.WithValue(ctx, NCLUSTERKEY, cluster)
}

// GetCtxTargetMethod
func GetCtxTargetMethod(ctx context.Context) (string, bool) {
	return getStrCtx(ctx, NMNAMEKEY)
}

// NewCtxWithTargetMethod
func NewCtxWithTargetMethod(ctx context.Context, method string) context.Context {
	return context.WithValue(ctx, NMNAMEKEY, method)
}

// GetCtxWithDefault return val from ctx, if the value is not exist or is "", return default val.
func GetCtxWithDefault(f func(ctx context.Context) (string, bool), ctx context.Context, val string) string {
	v, ok := f(ctx)
	if !ok || v == "" {
		return val
	}
	return v
}

// NewCtxWithConnectRetryMaxTime ...
func NewCtxWithConnectRetryMaxTime(ctx context.Context, timeout time.Duration) context.Context {
	return context.WithValue(ctx, CONNECTRETRYMAXTIMEKEY, timeout)
}

// GetCtxConnectRetryMaxTime ...
func GetCtxConnectRetryMaxTime(ctx context.Context) (time.Duration, bool) {
	if ctx == nil {
		return 0, false
	}
	v, ok := ctx.Value(CONNECTRETRYMAXTIMEKEY).(time.Duration)
	if !ok {
		return 0, false
	}
	return v, true
}

type TPolicy struct {
	IDC     string
	Percent int64 // total is 100
}

func NewCtxWithTPolicy(ctx context.Context, policy []TPolicy) context.Context {
	return context.WithValue(ctx, TRAFFICPOLICYKEY, policy)
}

func GetCtxTPolicy(ctx context.Context) ([]TPolicy, bool) {
	if ctx == nil {
		return nil, false
	}

	v, ok := ctx.Value(TRAFFICPOLICYKEY).([]TPolicy)
	if !ok {
		return nil, false
	}
	return v, true
}

// NewCtxRetryTimes set the retry times for a RPC call from context
func NewCtxWithRetryTimes(ctx context.Context, retry int) context.Context {
	return context.WithValue(ctx, RETRYTIMES, retry)
}

// GetCtxRetryTimes get the retry times for a RPC call from context
func GetCtxRetryTimes(ctx context.Context) (int, bool) {
	if ctx == nil {
		return 0, false
	}
	v, ok := ctx.Value(RETRYTIMES).(int)
	if !ok {
		return 0, false
	}
	return v, true
}

// Instance 代表一台下游的机器实例
type Instance interface {
	Host() string
	Port() string
	Tags() map[string]string
}

func NewCtxWithInstances(ctx context.Context, instances []Instance) context.Context {
	return context.WithValue(ctx, INSTANCESKEY, instances)
}

func GetCtxInstances(ctx context.Context) ([]Instance, bool) {
	if ctx == nil {
		return nil, false
	}

	v, ok := ctx.Value(INSTANCESKEY).([]Instance)
	if !ok {
		return nil, false
	}
	return v, true
}

// NewCtxWithTargetInstance .
func NewCtxWithTargetInstance(ctx context.Context, instance Instance) context.Context {
	return context.WithValue(ctx, TARGET_INSTANCE_KEY, instance)
}

// GetCtxTargetInstance .
func GetCtxTargetInstance(ctx context.Context) (Instance, bool) {
	if ctx == nil {
		return nil, false
	}

	v, ok := ctx.Value(TARGET_INSTANCE_KEY).(Instance)
	if !ok {
		return nil, false
	}
	return v, true
}

// NewCtxWithDegradationPercent .
func NewCtxWithDegradationPercent(ctx context.Context, sw int) context.Context {
	return context.WithValue(ctx, DEGRADATION_PERCENT_KEY, sw)
}

// GetCtxDegradationPercent .
func GetCtxDegradationPercent(ctx context.Context) (int, bool) {
	if ctx == nil {
		return 0, false
	}

	v, ok := ctx.Value(DEGRADATION_PERCENT_KEY).(int)
	if !ok {
		return 0, false
	}
	return v, true
}

// NewCtxWithTargetConn .
func NewCtxWithTargetConn(ctx context.Context, conn net.Conn) context.Context {
	return context.WithValue(ctx, TARGET_CONNECTION_KEY, conn)
}

// GetCtxTargetConn .
func GetCtxTargetConn(ctx context.Context) (net.Conn, bool) {
	if ctx == nil {
		return nil, false
	}

	v, ok := ctx.Value(TARGET_CONNECTION_KEY).(net.Conn)
	if !ok {
		return nil, false
	}
	return v, true
}

// NewCtxWithTargetConn .
func NewCtxWithRPCTimeout(ctx context.Context, timeout time.Duration) context.Context {
	return context.WithValue(ctx, RPC_TIMEOUT_KEY, timeout)
}

// GetCtxRPCTimeout .
func GetCtxRPCTimeout(ctx context.Context) (time.Duration, bool) {
	if ctx == nil {
		return 0, false
	}

	v, ok := ctx.Value(RPC_TIMEOUT_KEY).(time.Duration)
	if !ok {
		return 0, false
	}
	return v, true
}

// NewCtxWithAutoRPCRetry .
func NewCtxWithAutoRPCRetry(ctx context.Context, retry bool) context.Context {
	return context.WithValue(ctx, AUTO_RPC_RETRY_KEY, retry)
}

// GetCtxAutoRPCRetry .
func GetCtxAutoRPCRetry(ctx context.Context) (bool, bool) {
	if ctx == nil {
		return false, false
	}

	v, ok := ctx.Value(AUTO_RPC_RETRY_KEY).(bool)
	if !ok {
		return false, false
	}
	return v, true
}

// NewCtxWithBreakerSwitch .
func NewCtxWithBreakerSwitch(ctx context.Context, useCB bool) context.Context {
	return context.WithValue(ctx, BREAKER_SWITCH_KEY, useCB)
}

// GetCtxBreakerSwitch .
func GetCtxBreakerSwitch(ctx context.Context) (bool, bool) {
	if ctx == nil {
		return false, false
	}

	v, ok := ctx.Value(BREAKER_SWITCH_KEY).(bool)
	if !ok {
		return false, false
	}
	return v, true
}

// CallTuple contains five members in a call context
type CallTuple struct {
	From        string
	FromCluster string
	To          string
	ToCluster   string
	Method      string
}

// NewCtxWithCallTuple create a CallTuple
func NewCtxWithCallTuple(ctx context.Context, ct *CallTuple) context.Context {
	return context.WithValue(ctx, CallTupleKey{}, ct)
}

// GetCtxCallTuple return a *CallTuple instance from ctx
func GetCtxCallTuple(ctx context.Context) (*CallTuple, bool) {
	if ctx == nil {
		return nil, false
	}
	v, ok := ctx.Value(CallTupleKey{}).(*CallTuple)
	if !ok {
		return nil, false
	}
	return v, true
}

type CBConfig struct {
	ErrRate        float64
	MinSample      int
	MaxConcurrency int
}

// NewCtxWithCBConfig write retry conf into context
func NewCtxWithCBConfig(ctx context.Context, cnf *CBConfig) context.Context {
	return context.WithValue(ctx, CBConfigKey{}, cnf)
}

// GetCtxCBConfig read circuitbreaker config from context
func GetCtxCBConfig(ctx context.Context) (*CBConfig, bool) {
	if ctx == nil {
		return nil, false
	}
	cnf, ok := ctx.Value(CBConfigKey{}).(*CBConfig)
	if !ok {
		return nil, false
	}
	return cnf, true
}

type RetryConfig struct {
	LimitRate     float64
	MaxRetryTimes int
}

// NewRetryConfig write retry config into context
func NewCtxWithRetryConfig(ctx context.Context, cnf *RetryConfig) context.Context {
	return context.WithValue(ctx, RetryConfigKey{}, cnf)
}

// GetCtxRetryConfig read retry config from context
func GetCtxRetryConfig(ctx context.Context) (*RetryConfig, bool) {
	if ctx == nil {
		return nil, false
	}
	cnf, ok := ctx.Value(RetryConfigKey{}).(*RetryConfig)
	if !ok {
		return nil, false
	}
	return cnf, true
}

func NewCtxWithConstIDC(ctx context.Context, idc string) context.Context {
	return context.WithValue(ctx, ConstIDCKey{}, idc)
}

func GetCtxConstIDC(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	idc, ok := ctx.Value(ConstIDCKey{}).(string)
	if !ok {
		return "", false
	}
	return idc, true
}

// NewCtxWithRIPFunc .
func NewCtxWithRIPFunc(ctx context.Context, ripFunc func(rip string)) context.Context {
	return context.WithValue(ctx, RIPFuncKey{}, ripFunc)
}

// GetCtxRIPFunc .
func GetCtxRIPFunc(ctx context.Context) (func(rip string), bool) {
	if ctx == nil {
		return nil, false
	}

	ripFunc, ok := ctx.Value(RIPFuncKey{}).(func(string))
	return ripFunc, ok
}
