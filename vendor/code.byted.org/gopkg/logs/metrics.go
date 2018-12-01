package logs

import (
	"fmt"
	"os"
	"strings"

	"code.byted.org/gopkg/metrics"
)

var (
	metricsClient *metrics.MetricsClient

	metricsTagWarn  = map[string]string{"level": "WARNING"}
	metricsTagError = map[string]string{"level": "ERROR"}
	metricsTagFatal = map[string]string{"level": "CRITICAL"} // 和py统一, 将fatal打成critical
	metricsLim      = 4                                      //  只打Warn及以上的日志,
)

func init() {
	// loadServicePSM is the service psm read from load.sh
	loadServicePSM = strings.TrimSpace(loadServicePSM)
	if len(loadServicePSM) > 0 {
		metricsClient = metrics.NewDefaultMetricsClient("toutiao.service.log", true)
		fmt.Fprint(os.Stdout, "Log metrics: toutiao.service.log."+loadServicePSM+".throughput")
	}
}

func doMetrics(logLevel int) {
	if metricsClient == nil {
		return
	}
	if logLevel < metricsLim {
		return
	}

	if logLevel == 4 { // warning
		metricsClient.EmitCounter(loadServicePSM+".throughput", 1, "", metricsTagWarn)
	} else if logLevel == 5 { // error
		metricsClient.EmitCounter(loadServicePSM+".throughput", 1, "", metricsTagError)
	} else if logLevel == 6 { // fatal
		metricsClient.EmitCounter(loadServicePSM+".throughput", 1, "", metricsTagFatal)
	}
}
