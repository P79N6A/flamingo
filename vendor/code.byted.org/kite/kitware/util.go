package kitware

import (
	"strings"
	"sync"

	"code.byted.org/gopkg/net2"
	"code.byted.org/kite/endpoint"
	"code.byted.org/kite/kitutil"
)

var (
	onceLocalIP = &sync.Once{}
	localIP     string
)

func getRespCode(resp interface{}) (int, bool) {
	kitResp, ok := resp.(endpoint.KitcCallResponse)
	if ok == false { // if resp is invalid, return directly
		return 0, false
	}

	baseResp := kitResp.GetBaseResp()
	if baseResp == nil {
		return 0, false
	}

	code := baseResp.GetStatusCode()
	return int(code), true
}

func LocalIP() string {
	onceLocalIP.Do(func() {
		localIP = net2.GetLocalIp()
	})
	return localIP
}

// EtcdKeyJoin
func EtcdKeyJoin(a []string) string {
	if len(a) == 0 {
		return ""
	}
	n := 0
	for _, s := range a {
		n += len(s)
	}
	b := make([]byte, 0, 2*n)
	for _, s := range a {
		s = strings.TrimSpace(s)
		if len(s) == 0 {
			continue
		}
		if s[0] != '/' {
			b = append(b, '/')
		}
		b = append(b, s...)
	}
	return string(b)
}

type CallTupleKey struct {
	Prefix string
	kitutil.CallTuple
	Suffix string
}

var (
	ctpMutex sync.RWMutex
	ctpCache = make(map[CallTupleKey]string)
)

func EtcdCallTuplePropKey(ctp CallTupleKey) string {
	ctpMutex.RLock()
	s, ok := ctpCache[ctp]
	ctpMutex.RUnlock()
	if ok {
		return s
	}
	ctpMutex.Lock()
	if s, ok := ctpCache[ctp]; ok {
		ctpMutex.Unlock()
		return s
	}
	var ss = make([]string, 0, 16)
	if ctp.Prefix != "" {
		ss = append(ss, ctp.Prefix)
	}
	ss = append(ss, ctp.From)
	if ctp.FromCluster != DEFAULT_CLUSTER {
		ss = append(ss, ctp.FromCluster)
	}
	ss = append(ss, ctp.To)
	if ctp.ToCluster != DEFAULT_CLUSTER {
		ss = append(ss, ctp.ToCluster)
	}
	ss = append(ss, ctp.Method)
	if ctp.Suffix != "" {
		ss = append(ss, ctp.Suffix)
	}
	s = EtcdKeyJoin(ss)
	ctpCache[ctp] = s
	ctpMutex.Unlock()
	return s
}
