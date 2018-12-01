package kitc

import (
	"code.byted.org/kite/endpoint"
	"code.byted.org/kite/kitware"
)

var (
	isKiteService      = false
	kiteServiceName    = ""
	kiteServiceCluster = ""
)

// SetKiteService lets kitc know that it's used in a kite service and
// the kite service's name and cluster;
// More detail: https://code.byted.org/kite/kitc/issues/50.
func SetKiteService(name, cluster string) {
	isKiteService = true
	kiteServiceName = name
	kiteServiceCluster = cluster
}

func kitcChain(kc *KitcClient, opts *options) endpoint.Middleware {
	mids := []endpoint.Middleware{
		// write base members into context
		kitware.NewBaseWriterMW(),
		// print RPC logging
		kitware.NewRPCLogMW(logger),
		// emit RPC metrics
		kitware.NewRPCMetricsMW(metricsClient),
		// service breaker
		kitware.NewServiceBreakerMW(kc.circuitBreaker),
		// read acl config from remote to control this RPC
		kitware.NewRPCACLMW(NewAcler(etcdCache)),
		// read dynamic config and write into context
		kitware.NewDynamicConfigMW(NewDynamicClient(etcdCache)),
		// service degradation middleware
		kitware.NewDegradationMW(NewDegradater(etcdCache)),
		// select IDC
		kitware.NewDefaultIDCSelectorMW(LocalIDC()),
		// service discover
		kitware.NewServiceDiscoverMW(kc.discoverer),
		// LB and RPC retry
		kitware.NewConnRetryMW(connBalanceRetrier),
		// Instance CB
		kitware.NewInstanceBreakerMW(kc.instanceBreaker),
		// Pool
		kitware.NewPoolMW(kc.Pool),
		// I/O error handler
		kitware.NewIOErrorHandlerMW(),
	}

	return endpoint.Chain(mids[0], mids[1:]...)
}
