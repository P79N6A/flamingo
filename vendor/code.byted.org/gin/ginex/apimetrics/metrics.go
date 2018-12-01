package apimetrics

import (
	"fmt"
	"strconv"
	"time"

	"code.byted.org/gin/ginex/internal"
	"code.byted.org/gopkg/metrics"
	"github.com/gin-gonic/gin"
)

const (
	METRICS_PREFIX           = "toutiao.service.http"
	FRAMEWORK_METRICS_PREFIX = "toutiao.service"
)

func Metrics(psm string) gin.HandlerFunc {
	emitter := metrics.NewDefaultMetricsClient(METRICS_PREFIX, true)
	latencyMetrics := fmt.Sprintf("%s.calledby.success.latency.us", psm)
	throughputMetrics := fmt.Sprintf("%s.calledby.success.throughput", psm)
	frameWorkThroughputMetrics := "ginex.throughput"
	frameWorkMetricsTags := map[string]string{
		"version": internal.VERSION,
	}
	return func(c *gin.Context) {
		if psm == "" {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()
		end := time.Now()
		latency := end.Sub(start).Nanoseconds() / 1000
		var handleMethod string
		if v := c.Value(internal.METHODKEY); v != nil {
			// 如果METHODKEY设置了,那它一定是string
			handleMethod = v.(string)
		}
		tags := map[string]string{
			"status":        strconv.Itoa(c.Writer.Status()),
			"handle_method": handleMethod,
			"from_cluster":  "default",
			"to_cluster":    "default",
		}
		// https://wiki.bytedance.net/pages/viewpage.action?pageId=51348664
		emitter.EmitTimer(latencyMetrics, latency, "", tags)
		emitter.EmitCounter(throughputMetrics, 1, "", tags)
		// emit framework metrics
		emitter.EmitCounter(frameWorkThroughputMetrics, 1, FRAMEWORK_METRICS_PREFIX, frameWorkMetricsTags)
	}
}
