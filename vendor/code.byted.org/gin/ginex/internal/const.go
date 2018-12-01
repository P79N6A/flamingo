package internal

const (
	TT_LOGID_HEADER_KEY          = "X-TT-LOGID" // Http header中log id key
	TT_LOGID_HEADER_FALLBACK_KEY = "X-Tt-Logid" // unknown fallback
	LOGIDKEY                     = "K_LOGID"    // 唯一的Request ID
	SNAMEKEY                     = "K_SNAME"    // 本服务的名字
	LOCALIPKEY                   = "K_LOCALIP"  // 本服务的IP 地址
	CLUSTERKEY                   = "K_CLUSTER"  // 本服务集群的名字
	METHODKEY                    = "K_METHOD"   // 本服务当前所处的接口名字（也就是Method名字）

	HOST_IP_ADDR    = "HOST_IP_ADDR"
	TCE_CLUSTER     = "TCE_CLUSTER"
	SERVICE_CLUSTER = "SERVICE_CLUSTER"
)

// Envs set by ginex, begins with "_GINEX"
const (
	GINEX_PSM = "_GINEX_PSM"
)

// Ginex framework version
const (
	VERSION = "v1.1.1"
)
