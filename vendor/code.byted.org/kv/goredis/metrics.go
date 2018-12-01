package goredis

import (
	"os"
	"strings"

	"code.byted.org/gopkg/metrics"
)

const (
	CALLSTATUS_SUCCESS = "success"
	CALLSTATUS_MISS    = "miss"
	CALLSTATUS_ERROR   = "error"
)

const (
	METRICSNAME_PREFIX = "toutiao.service.thrift."
	REDIS_PSM_PREFIX   = "toutiao.redis."
	REDIS_PREFIX       = "redis_"
	ABASE_PSM_PREFIX   = "toutiao.abase."
	ABASE_PREFIX       = "abase_"
)

var metricsClient *metrics.MetricsClient
var metricsClientWithPsm *metrics.MetricsClient

func init() {
	metricsClient = metrics.NewDefaultMetricsClient("redis.client", true)
	metricsClient.DefineCounter("throughput", "")
	metricsClient.DefineCounter("error", "")
	metricsClient.DefineCounter("miss", "")
	metricsClient.DefineTimer("latency", "") //us

	metricsClientWithPsm = metrics.NewDefaultMetricsClient(METRICSNAME_PREFIX+checkPsm(), true)
	metricsClientWithPsm.DefineCounter("call.success.throughput", "")
	metricsClientWithPsm.DefineCounter("call.error.throughput", "")
	metricsClientWithPsm.DefineCounter("call.miss.throughput", "")
	metricsClientWithPsm.DefineTimer("call.success.latency.us", "")
	metricsClientWithPsm.DefineTimer("call.error.latency.us", "")
	metricsClientWithPsm.DefineTimer("call.miss.latency.us", "")
}

func addCallMetrics(
	cmd string,
	latency int64,
	status string,
	cluster string,
	psm string,
	metricsServiceName string) {
	// old
	tags := map[string]string{
		"cluster": cluster,
		"caller":  psm,
		"cmd":     cmd,
		"lang":    "go"}
	metricsClient.EmitTimer("latency", latency, "", tags)
	metricsClient.EmitCounter("throughput", 1, "", tags)
	metricsClient.EmitCounter(status, 1, "", tags)

	// new
	tagsForPsmClient := map[string]string{
		"cluster":      cluster,
		"method":       cmd,
		"to":           metricsServiceName,
		"from_cluster": "default",
		"to_cluster":   "default"}
	metricsClientWithPsm.EmitCounter("call."+status+".throughput", 1, "", tagsForPsmClient)
	metricsClientWithPsm.EmitTimer("call."+status+".latency.us", latency, "", tagsForPsmClient)
}

// TODO update DC info
func getDcName(ip string) string {
	if ip == "" {
		return "None"
	} else {
		if strings.HasPrefix(ip, "10.4.") {
			return "hy"
		} else if strings.HasPrefix(ip, "10.6.") || strings.HasPrefix(ip, "10.3.") {
			return "lf"
		} else {
			return "Unidentified"
		}
	}
}

func checkPsm() string {
	psm := os.Getenv("TCE_PSM")
	if len(psm) == 0 {
		psm = os.Getenv("PSM")
	}
	if len(psm) == 0 {
		psm = os.Getenv("SVC_NAME")
	}
	if len(psm) == 0 {
		psm = "redis.psm.none"
	}
	return psm
}

// return redis_XXX / abase_XXX
func GetClusterName(str string) string {
	clusterName := str
	if strings.HasPrefix(str, REDIS_PSM_PREFIX) {
		clusterName = REDIS_PREFIX + clusterName[len(REDIS_PSM_PREFIX):]
	} else if strings.HasPrefix(str, ABASE_PSM_PREFIX) {
		clusterName = ABASE_PREFIX + clusterName[len(ABASE_PSM_PREFIX):]
	}
	return clusterName
}

// return toutiao.redis.XXX / toutiao.abase.XXX
func GetPSMClusterName(str string) string {
	PSMClusterName := str
	if strings.HasPrefix(str, REDIS_PREFIX) {
		PSMClusterName = REDIS_PSM_PREFIX + PSMClusterName[len(REDIS_PREFIX):]
	} else if strings.HasPrefix(str, ABASE_PREFIX) {
		PSMClusterName = ABASE_PSM_PREFIX + PSMClusterName[len(ABASE_PREFIX):]
	}
	return PSMClusterName
}
