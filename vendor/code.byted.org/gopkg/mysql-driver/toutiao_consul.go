package mysql

import (
	"fmt"
	"os"
	"strings"
	"time"

	"code.byted.org/golf/consul"
	"code.byted.org/gopkg/asyncache"
)

/*
	There are four patterns of DSN in toutiao now:
		1) consul with prefix "consul:":
			USERNAME:PASSWORD@tcp(consul:toutiao.mysql.ms_data_write)/DATABASE
		2) consul without prefix:
			USERNAME:PASSWORD@tcp(toutiao.mysql.ms_data_write)/DATABASE
		3) multi-host DSN:
			USERNAME:PASSWORD@tcp(10.4.16.18:3306,127.0.0.1:3306)/DATABASE
		4) normal DSN(single host):
			USERNAME:PASSWORD@tcp(10.4.16.18:3306)/DATABASE
		5) multi-host-one-port:
			USERNAME:PASSWORD@tcp(10.4.16.18,127.0.0.1:3306)/DATABASE

	convertConsulDSN Convert pattern 1 and 2 to pattern 3 or 4, and return consulName;
*/
func convertConsulDSN(dsn string) (converedDSN string, consulName string) {
	originDSN := dsn

	hookTag := "@tcp("
	left := strings.Index(dsn, hookTag)
	if left == -1 {
		return originDSN, ""
	}

	left += len(hookTag)
	consulPrefix := "consul:"
	if strings.HasPrefix(dsn[left:], consulPrefix) {
		// pattern 1, remove prefix
		dsn = dsn[:left] + dsn[left+len(consulPrefix):]
	}

	right := strings.Index(dsn[left:], ")")
	if right == -1 {
		return originDSN, ""
	}
	right += left

	if isInvalidPSM(dsn[left:right]) == false {
		str := dsn[left:right]
		if len(strings.Split(str, ",")) > 1 && len(strings.Split(str, ":")) == 2 {
			// pattern 5, convert it to pattern 3
			tmp := strings.Split(str, ":")
			port := tmp[1]
			hosts := strings.Split(tmp[0], ",")
			addrs := make([]string, 0, len(hosts))
			for _, host := range hosts {
				addrs = append(addrs, fmt.Sprintf("%v:%v", host, port))
			}
			addrStr := strings.Join(addrs, ",")
			return dsn[:left] + addrStr + dsn[right:], ""
		}

		// pattern 3 or 4, return directly
		return originDSN, ""
	}

	consulName = dsn[left:right]
	var addrs []*consul.Endpoint
	var err error
	for try := 0; try < 3; try++ {
		addrs, err = translateOne(consulName)
		if err == nil {
			break
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "[mysql-driver]: consul translate %v err: %v\n", consulName, err)
		return originDSN, ""
	}

	addrList := make([]string, 0, len(addrs))
	for _, end := range addrs {
		if end.Host != "" {
			addrList = append(addrList, fmt.Sprintf("%v:%v", end.Host, end.Port))
		}
	}
	if len(addrList) == 0 {
		fmt.Fprintf(os.Stderr, "[mysql-driver]: no host found for consulName: %v\n", consulName)
		return originDSN, ""
	}

	addrsStr := strings.Join(addrList, ",")
	return dsn[:left] + addrsStr + dsn[right:], consulName
}

func isInvalidPSM(psm string) bool {
	return len(strings.Split(psm, ".")) == 3
}

func addrToConsulName(addr string) string {
	tmp := strings.Split(addr, ":")
	if len(tmp) != 2 {
		return addr
	}

	return strings.Replace(tmp[1], ".", "_", -1)
}

var consulCache *asyncache.SingleAsyncCache

// type Getter func(key string) (interface{}, error)
func init() {
	getter := func(key string) (interface{}, error) {
		eps, err := consul.TranslateOne(key)
		if err != nil {
			return nil, err
		}
		return eps, nil
	}
	consulCache = asyncache.NewBlockedAsyncCache(getter, time.Second)
}

func translateOne(consulName string) ([]*consul.Endpoint, error) {
	val, err := consulCache.Get(consulName)
	if err != nil {
		return nil, err
	}
	if eps, ok := val.([]*consul.Endpoint); ok {
		return eps, nil
	}
	return nil, fmt.Errorf("translateOne consulName err: invalid val type")
}
