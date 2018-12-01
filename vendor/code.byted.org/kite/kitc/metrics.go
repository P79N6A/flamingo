/*
RPC Call Metrics Format

toutiao.service.thrift.{PSM}.call.{PSM}.{method}.success.latency.us
toutiao.service.thrift.{PSM}.call.{PSM}.{method}.error.latency.us

toutiao.service.thrift.{PSM}.call.{PSM}.{method}.success.throughput
toutiao.service.thrift.{PSM}.call.{PSM}.{method}.error.throughput

*/
package kitc

import (
	"code.byted.org/gopkg/metrics"
)

var (
	metricsClient *metrics.MetricsClient
)

const (
	namespacePrefix string = "toutiao"
)

func init() {
	metricsClient = metrics.NewDefaultMetricsClient(namespacePrefix, true)
}
