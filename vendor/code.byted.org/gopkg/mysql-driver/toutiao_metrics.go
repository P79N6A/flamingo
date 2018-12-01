package mysql

import (
	"fmt"
	"strconv"
	"time"

	"code.byted.org/gopkg/metrics"
)

var (
	metricsCli = metrics.NewDefaultMetricsClient("toutiao.service.thrift", true)

	successRPCThroughputFmt = "%s.call.success.throughput"
	errorRPCThroughputFmt   = "%s.call.error.throughput"
	successRPCLatencyFmt    = "%s.call.success.latency.us"
	errorRPCLatencyFmt      = "%s.call.error.latency.us"
)

func doMetrics(sql string, cfg *Config, cost time.Duration, err error) {
	operation, _ := getOperation(sql)

	to := consulName2PSM(cfg.toutiaoConsulName)
	costInUS := int64(cost / 1000)
	tags := map[string]string{
		"to":           to,
		"method":       operation,
		"from_cluster": serviceCluster,
		"to_cluster":   "default",
		"table":        getTableName(operation, sql),
	}

	var throughputMetrics, latencyMetrics string
	errCode := getMysqlErrCode(err)
	if errCode == 0 {
		throughputMetrics = fmt.Sprintf(successRPCThroughputFmt, serviceName)
		latencyMetrics = fmt.Sprintf(successRPCLatencyFmt, serviceName)
	} else {
		throughputMetrics = fmt.Sprintf(errorRPCThroughputFmt, serviceName)
		latencyMetrics = fmt.Sprintf(errorRPCLatencyFmt, serviceName)
		tags["err_code"] = strconv.Itoa(errCode)
	}

	metricsCli.EmitCounter(throughputMetrics, 1, "", tags)
	metricsCli.EmitTimer(latencyMetrics, costInUS, "", tags)
}
