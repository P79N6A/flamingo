package env

import (
	"strings"
	"sync/atomic"
)

const (
	UnknownIDC = "-"
	DC_HY      = "hy"
	DC_LF      = "lf"
	DC_VA      = "va"
	DC_SG      = "sg"
	DC_CA      = "ca"    // West America
	DC_ALISG   = "alisg" // Singapore Aliyun
)

var (
	idc       atomic.Value
	idcPrefix = map[string][]string{
		DC_HY:    []string{"10.4."},
		DC_LF:    []string{"10.2.", "10.3.", "10.6.", "10.8.", "10.9.", "10.10.", "10.11.", "10.12."},
		DC_VA:    []string{"10.100."},
		DC_SG:    []string{"10.101."},
		DC_CA:    []string{"10.106."},
		DC_ALISG: []string{"10.115."},
	}
)

// IDC .
func IDC() string {
	if v := idc.Load(); v != nil {
		return v.(string)
	}

	ip := HostIP()
	for idcStr, pres := range idcPrefix {
		for _, p := range pres {
			if strings.HasPrefix(ip, p) {
				idc.Store(idcStr)
				return idcStr
			}
		}
	}

	idc.Store(UnknownIDC)
	return UnknownIDC
}
