package goclient

import (
	"code.byted.org/gopkg/metrics"
	"errors"
)

var SessionMetricsClient *metrics.MetricsClient

const (
	SESSION_METRICS_PREFIX = "web.passport.session"
	THROUGHPUT_SUFFIX      = ".throughput"
	LATENCY_SUFFIX         = ".latency"
	ERROR_SUFFIX           = ".error"
)

func init() {
	SessionMetricsClient = metrics.NewDefaultMetricsClient(SESSION_METRICS_PREFIX, true)
	SessionMetricsClient.DefineCounter(METRICS_INVALID_SESSION_KEY, "")
	SessionMetricsClient.DefineCounter(METRICS_UID_CHANGED, "")
}

func EmitCounter(name string, value interface{}, tagkv map[string]string) error {
	if SessionMetricsClient == nil {
		return errors.New("MetricsClient not init")
	}
	return SessionMetricsClient.EmitCounter(name+THROUGHPUT_SUFFIX, value, "", tagkv)
}

func EmitError(name string, value interface{}, tagkv map[string]string) error {
	if SessionMetricsClient == nil {
		return errors.New("MetricsClient not init")
	}
	return SessionMetricsClient.EmitCounter(name+ERROR_SUFFIX, value, "", tagkv)
}

func EmitTimer(name string, value interface{}, tagkv map[string]string) error {
	if SessionMetricsClient == nil {
		return errors.New("MetricsClient not init")
	}
	return SessionMetricsClient.EmitTimer(name+LATENCY_SUFFIX, value, "", tagkv)
}
