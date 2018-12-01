/*
Metrics 定义描述 https://wiki.bytedance.com/pages/viewpage.action?pageId=51348664
*/
package kite

import (
	"code.byted.org/gopkg/metrics"
	"code.byted.org/gopkg/stats"
)

const (
	namespacePrefix string = "toutiao"
)

var (
	metricsClient MetricsEmiter
)

func init() {
	// IgnoreCheck is true.
	metricsClient = metrics.NewDefaultMetricsClient(namespacePrefix, true)
}

// GoStatMetrics emit GC, Stask, Heap info to TSDB.
func GoStatMetrics() error {
	return stats.DoReport(ServiceName)
}

type MetricsEmiter interface {
	EmitCounter(name string, value interface{}, prefix string, tagkv map[string]string) error
	EmitTimer(name string, value interface{}, prefix string, tagkv map[string]string) error
	EmitStore(name string, value interface{}, prefix string, tagkv map[string]string) error
}

type EmptyEmiter struct{}

func (ee *EmptyEmiter) EmitCounter(name string, value interface{}, prefix string, tagkv map[string]string) error {
	return nil
}

func (ee *EmptyEmiter) EmitTimer(name string, value interface{}, prefix string, tagkv map[string]string) error {
	return nil
}

func (ee *EmptyEmiter) EmitStore(name string, value interface{}, prefix string, tagkv map[string]string) error {
	return nil

}
