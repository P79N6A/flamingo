package goclient

type SessionClientConfig struct {
	caller           string // 标识调用来源，优先使用P.S.M
	timeout          uint16 // 单位ms,不设置使用 DEFAULT_TIME_OUT
	connTimeout      uint16 // 单位ms,不设置使用 DEFAULT_CONN_TIME_OUT
	connMaxRetryTime uint16 // 单位ms,不设置使用 DEFAULT_CONN_MAX_RETRY_TIME
}

func NewSessionClientConfig(caller string, timeout uint16, connTimeout uint16, connMaxRetryTime uint16) SessionClientConfig {
	return SessionClientConfig{caller, timeout, connTimeout, connMaxRetryTime}
}
