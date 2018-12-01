package kitc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"code.byted.org/kite/endpoint"
	"code.byted.org/kite/kiterrno"
	"code.byted.org/kite/kitutil"
	"code.byted.org/kite/kitware"
)

// Client is a interface that every generate code should implement this
type Client interface {
	New(kc *KitcClient) Caller
}

// Caller is a interface for client do soem RPC calling
type Caller interface {
	Call(name string, request interface{}) (endpoint.EndPoint, endpoint.KitcCallRequest)
}

// client just is an instance have some methods, client will contains connections
var clients = make(map[string]Client)

// Register ...
func Register(name string, client Client) {
	if client == nil {
		panic("kitc: Register client is nil")
	}
	if _, dup := clients[name]; dup {
		panic("kitc: Register dup client")
	}
	clients[name] = client
}

// KitcClient ...
type KitcClient struct {
	name            string
	opts            *options
	client          Client
	circuitBreaker  *ServiceCircuit
	instanceBreaker *InstanceCircuit
	retryManager    *RetryManager
	lock            sync.RWMutex
	mids            [15]endpoint.Middleware
	Pool            ConnPool
	// service discover
	discoverer *Discoverer // service discover
}

// NewClient ...
func NewClient(name string, ops ...Option) (*KitcClient, error) {
	client, ok := clients[name]
	if !ok {
		return nil, Err("Unknow client name %q, forget import?", name)
	}
	opts := new(options)
	for _, do := range ops {
		do.f(opts)
	}

	var ikService IKService
	if opts.iks != nil {
		ikService = opts.iks
	} else if opts.insList != nil {
		ikService = NewCustomIKService(name, opts.insList)
	} else {
		ikService = NewConsulService(name)
	}

	if opts.useWatcher {
		ikService = NewConsulWatcherService(ikService)
	} else {
		ikService = NewCacheService(ikService)
	}

	if opts.CircuitBreaker.ErrorRate <= 0 {
		opts.CircuitBreaker.ErrorRate = DefaultErrRate
	}

	if opts.CircuitBreaker.MaxConcurrency <= 0 {
		opts.CircuitBreaker.MaxConcurrency = DefaultMaxConcurrency
	}

	if opts.CircuitBreaker.MinSamples <= 0 {
		opts.CircuitBreaker.MinSamples = DefaultMinSamples
	}

	breaker := NewServiceCircuit(opts.CircuitBreaker.ErrorRate, opts.CircuitBreaker.MaxConcurrency, opts.CircuitBreaker.MinSamples)
	try := func(ret interface{}, err error) bool {
		if err == nil {
			return false
		}
		if ret == nil {
			return false
		}
		resp, ok := ret.(endpoint.KitcCallResponse)
		if !ok {
			return false
		}
		base := resp.GetBaseResp()
		if base == nil {
			return false
		}
		code := base.GetStatusCode()
		switch int(code) {
		case kiterrno.NotAllowedByInstanceCBCode,
			kiterrno.NotAllowedByServiceCBCode,
			kiterrno.ForbiddenByDegradationCode,
			kiterrno.NotAllowedByACLCode:
			return false
		}
		return true
	}

	kitclient := &KitcClient{
		name:            name,
		opts:            opts,
		client:          client,
		instanceBreaker: NewInstanceCircuit(),
		circuitBreaker:  breaker,
		retryManager:    NewRetryManager(try),
		Pool:            NewPool(name, ops...),
		discoverer:      NewDiscoverer(name, ikService),
		lock:            sync.RWMutex{},
	}
	kitclient.mids = [15]endpoint.Middleware{
		// write base members into context
		kitware.NewBaseWriterMW(),
		// RPC logs
		nil, // init in the first Call()
		// emit RPC metrics
		kitware.NewRPCMetricsMW(metricsClient),
		// read dynamic config and write into context
		kitware.NewDynamicConfigMW(NewDynamicClient(etcdCache)),
		// config reporter MW
		nil, // init in doReport()
		// service breaker config
		kitware.NewServiceCircuitBreakerConfigMW(NewServiceCircuitConfig(etcdCache)),
		// service breaker
		kitware.NewServiceBreakerMW(kitclient.circuitBreaker),
		// read acl config from remote to control this RPC
		kitware.NewRPCACLMW(NewAcler(etcdCache)),
		// service degradation middleware
		kitware.NewDegradationMW(NewDegradater(etcdCache)),
		// select IDC
		kitware.NewDefaultIDCSelectorMW(LocalIDC()),
		// service discover
		kitware.NewServiceDiscoverMW(kitclient.discoverer),
		// LB and RPC retry
		kitware.NewConnRetryMW(connBalanceRetrier),
		// Instance CB
		kitware.NewInstanceBreakerMW(kitclient.instanceBreaker),
		// Pool
		kitware.NewPoolMW(kitclient.Pool),
		// I/O error handler
		kitware.NewIOErrorHandlerMW(),
	}

	kitclient.doReport()

	return kitclient, nil
}

// doReport reports config in this client
func (kc *KitcClient) doReport() {
	if isKiteService { // report runtime timeout config
		reportInterval := time.Second * 15
		lastTimestamp := make(map[string]time.Time) // [to:to_cluster:method]last_report_time
		lastTimestampMutex := sync.RWMutex{}
		type TimeoutConf struct {
			To          string `json:"to"`
			ToCluster   string `json:"to_cluster"`
			Method      string `json:"method"`
			RPCTimeout  int    `json:"rpc_timeout"`  // ms
			ConnTimeout int    `json:"conn_timeout"` // ms
			RetryTimes  int    `json:"retry_times"`
		}
		kc.mids[4] = kitware.NewConfigReporterMW(func(ctx context.Context) {
			tuple, ok := kitutil.GetCtxCallTuple(ctx)
			if !ok {
				return
			}
			key := fmt.Sprintf("%v:%v:%v", tuple.To, tuple.ToCluster, tuple.Method)

			lastTimestampMutex.RLock()
			last, ok := lastTimestamp[key]
			lastTimestampMutex.RUnlock()
			if ok && last.Add(reportInterval).After(time.Now()) {
				return // not expire now
			}

			lastTimestampMutex.Lock()
			if _, ok := lastTimestamp[key]; ok {
				lastTimestampMutex.Unlock()
				return // being reported by other goroutine now
			}
			lastTimestamp[key] = time.Now()
			lastTimestampMutex.Unlock()

			go func() {
				rpcTimeout, _ := kitutil.GetCtxRPCTimeout(ctx)
				connTimeout := kitware.DefaultConnTimeout // always use default connTimeout
				retryTimes, _ := kitutil.GetCtxRetryTimes(ctx)
				conf := &TimeoutConf{
					To:          tuple.To,
					ToCluster:   tuple.ToCluster,
					Method:      tuple.Method,
					RPCTimeout:  int(rpcTimeout / time.Millisecond),
					ConnTimeout: int(connTimeout / time.Millisecond),
					RetryTimes:  retryTimes,
				}
				if buf, err := json.Marshal(conf); err == nil {
					reporter.Report("msframe:config", "timeout", buf)
				}
			}()
		})
	} else {
		kc.mids[4] = kitware.NewConfigReporterMW(func(context.Context) {
			return // do nothing if not in kite service
		})
	}

	if !isKiteService {
		return
	}

	// report circuitbreaker config every second
	go func() {
		<-canUseRepoter
		for range time.Tick(time.Second) {
			metrics := kc.circuitBreaker.AllMetrics()
			data, _ := json.Marshal(metrics)
			reporter.Report("kite:circuitbreaker", "circuitbreaker", data)
		}
	}()

	// report long connection pool config every 30 seconds
	if long, ok := kc.Pool.(*LongPool); ok {
		type LongConnConf struct {
			To             string `json:"to"`
			ToCluster      string `json:"to_cluster"`
			MaxIdle        int    `json:"max_idle"`
			MaxIdleTimeout int    `json:"max_idle_timeout"` // in MS
		}
		conf := &LongConnConf{
			To:             kc.name,
			ToCluster:      kc.opts.cluster,
			MaxIdle:        long.maxIdleConns,
			MaxIdleTimeout: int(long.maxIdleTimeout / time.Millisecond),
		}
		if conf.ToCluster == "" {
			conf.ToCluster = "default"
		}
		data, _ := json.Marshal(conf)
		go func() {
			<-canUseRepoter
			for range time.Tick(time.Second * 30) {
				reporter.Report("msframe:config", "conn_pool", data)
			}
		}()
	}
}

// Call do some remote calling
func (kc *KitcClient) Call(name string, ctx context.Context, request interface{}) (endpoint.KitcCallResponse, error) {
	metricsClient.EmitCounter("kite.request.throughput", 1, "", nil)

	ctx, err := kc.initContext(name, ctx)
	if err != nil {
		return nil, err
	}
	key, err := kc.retryManager.RetryKey(ctx)
	if err != nil {
		return nil, err
	}

	var (
		limitRate float64 = DefaultLimitRetryRate
		maxRetry  int     = DefaultMaxRetryTimes
	)
	retryCnf, ok := kitutil.GetCtxRetryConfig(ctx)
	if ok {
		limitRate = retryCnf.LimitRate
		maxRetry = retryCnf.MaxRetryTimes
	}
	retrier := kc.retryManager.Retry(key)
	resp, err := retrier.Do(limitRate, maxRetry, func() (interface{}, error) {
		caller := kc.client.New(kc)
		next, request := caller.Call(name, request)
		if next == nil || request == nil {
			return nil, Err("service=%s method=%s  unknow method return nil", kc.name, name)
		}
		kc.lock.RLock()
		mids := kc.mids
		kc.lock.RUnlock()
		if mids[1] == nil {
			if logger == nil {
				mids[1] = kitware.NewRPCLogMW(&localLogger{})
			} else {
				mids[1] = kitware.NewRPCLogMW(logger)
				kc.lock.Lock()
				kc.mids[1] = mids[1]
				kc.lock.Unlock()
			}
		}

		chain := endpoint.Chain(mids[0], mids[1:]...)
		return chain(next)(ctx, request)
	})

	if resp == nil {
		return nil, err
	}
	return resp.(endpoint.KitcCallResponse), err
}

func (kc *KitcClient) initContext(method string, ctx context.Context) (context.Context, error) {
	ctx = kitutil.NewCtxWithTargetServiceName(ctx, kc.name)
	ctx = kitutil.NewCtxWithTargetMethod(ctx, method)
	ctx = kitutil.NewCtxWithTargetClusterName(ctx, kc.opts.cluster)
	fromCluster := kitutil.GetCtxWithDefault(kitutil.GetCtxCluster, ctx, "")
	from := kitutil.GetCtxWithDefault(kitutil.GetCtxServiceName, ctx, "")
	if from == "" {
		return nil, errors.New("no service's name for loanrpc call, you can use kitutil.NewCtxWithServiceName(ctx, xxxx) to set ctx")
	}
	ctx = kitutil.NewCtxWithCallTuple(ctx, &kitutil.CallTuple{
		From:        from,
		FromCluster: fromCluster,
		To:          kc.name,
		ToCluster:   kc.opts.cluster,
		Method:      method,
	})

	if kc.opts.idc != "" {
		ctx = kitutil.NewCtxWithConstIDC(ctx, kc.opts.idc)
	}

	if kc.opts.rpcTimeout != 0 {
		ctx = kitutil.NewCtxWithRPCTimeout(ctx, kc.opts.rpcTimeout)
	}

	if kc.opts.connMaxRetryTime != 0 {
		ctx = kitutil.NewCtxWithConnectRetryMaxTime(ctx, kc.opts.connMaxRetryTime)
	}

	if kc.opts.disableCB {
		ctx = kitutil.NewCtxWithBreakerSwitch(ctx, false)
		return ctx, nil
	}

	// config by this method calling
	_, ok := kitutil.GetCtxCBConfig(ctx)
	if ok {
		ctx = kitutil.NewCtxWithBreakerSwitch(ctx, true)
	}

	return ctx, nil
}

func (kc *KitcClient) Name() string {
	return kc.name
}
