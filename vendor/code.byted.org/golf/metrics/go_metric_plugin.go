package metrics

import (
	"errors"
	gometric "github.com/rcrowley/go-metrics"
	"strconv"
	"time"
)

var enableRuntimeMetric bool = false

type MetricConfig struct {
	registry      gometric.Registry
	flushInterval time.Duration
	prefix        string
}

func EnableGoRuntimeMetric(prefix string) error {
	if enableRuntimeMetric {
		return nil
	}
	status := IsInitialized()
	if !status {
		return errors.New("you must initialize metric instance first")
	}
	enableRuntimeMetric = true

	if prefix == "" {
		prefix = "inf.go.runtime"
	}

	config := &MetricConfig{
		registry:      gometric.DefaultRegistry,
		flushInterval: 10 * time.Second,
		prefix:        prefix,
	}
	go captureRuntimeMetric(config)
	return nil
}

func captureRuntimeMetric(config *MetricConfig) {
	gometric.RegisterRuntimeMemStats(config.registry)
	for _ = range time.Tick(config.flushInterval) {
		gometric.CaptureRuntimeMemStatsOnce(config.registry)
		runtimeMetricEmit(config)
	}
}

func runtimeMetricEmit(config *MetricConfig) {
	var tagkv map[string]string
	config.registry.Each(func(name string, i interface{}) {
		switch mt := i.(type) {
		case gometric.Counter:
			value := mt.Count()
			emitWithCheck("counter", name, strconv.FormatInt(value, 10), config.prefix, tagkv, false)
		case gometric.Gauge:
			value := mt.Value()
			emitWithCheck("store", name, strconv.FormatInt(value, 10), config.prefix, tagkv, false)
		case gometric.GaugeFloat64:
			value := mt.Value()
			emitWithCheck("store", name, strconv.FormatFloat(value, 'E', -1, 64), config.prefix, tagkv, false)
		case gometric.Histogram:
			value := mt.Mean()
			emitWithCheck("store", name, strconv.FormatFloat(value, 'E', -1, 64), config.prefix, tagkv, false)
		}
	})
}
