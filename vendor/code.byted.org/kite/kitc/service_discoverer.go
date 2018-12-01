package kitc

import (
	"code.byted.org/gopkg/logs"
	"code.byted.org/kite/kitutil"
)

// Discoverer for kitware
type Discoverer struct {
	name     string // for log
	ikserver IKService
}

// NewDiscoverer .
func NewDiscoverer(name string, ikserver IKService) *Discoverer {
	return &Discoverer{
		name,
		ikserver,
	}
}

// failback is hard code for balance hy and lf
// TODO(xiangchao.01)
func failbackDC(dc string) string {
	if dc == "lf" {
		return "hy"
	} else {
		return "lf"
	}
}

func (d *Discoverer) Name() string {
	return d.ikserver.Name()
}

func (d *Discoverer) Lookup(idc, targetCluster, upstreamEnv string) ([]kitutil.Instance, error) {
	addrs := d.ikserver.Lookup(idc)
	if len(addrs) == 0 {
		logs.Warn("psm: %v, cluster: %v, lookup dc: %s have no addrs", d.name, targetCluster, idc)
		idc = failbackDC(idc)
		logs.Warn("failback to dc: %s", idc)
		addrs = d.ikserver.Lookup(idc)
		if len(addrs) == 0 {
			return nil, Err("both lf and hy have no addrs")
		}
	}
	newAddrs := make([]*Instance, len(addrs))
	copy(newAddrs, addrs)
	addrs = newAddrs

	// 如果期待的cluster为空字符串，则可以匹配到下游cluster为空的，或者为default的实例
	isExpect := func(c string, expect, actual string) bool {
		if expect == actual {
			return true
		}
		if c == "cluster" {
			if expect == "" && actual == "default" {
				return true
			}
		}
		if c == "env" {
			if expect == "" && actual == "prod" {
				return true
			}
			// flow to canary env
			if expect == "" && actual == "canary" {
				return true
			}
			if expect == "prod" && actual == "canary" {
				return true
			}
		}
		return false
	}

	i, j := 0, len(addrs)-1
	for i < j {
		for isExpect("cluster", targetCluster, addrs[i].Cluster()) && i < j {
			i++
		}
		for !isExpect("cluster", targetCluster, addrs[j].Cluster()) && i < j {
			j--
		}
		addrs[i], addrs[j] = addrs[j], addrs[i]
	}

	if isExpect("cluster", targetCluster, addrs[i].Cluster()) {
		i += 1
	}
	addrs = addrs[:i]

	if len(addrs) == 0 {
		return nil, Err("No hosts left for cluster:%s", targetCluster)
	}

	// 调用下游服务的指定环境的实例
	i, j = 0, len(addrs)-1
	for i < j {
		for isExpect("env", upstreamEnv, addrs[i].Env()) && i < j {
			i++
		}
		for !isExpect("env", upstreamEnv, addrs[j].Env()) && i < j {
			j--
		}
		addrs[i], addrs[j] = addrs[j], addrs[i]
	}
	if isExpect("env", upstreamEnv, addrs[i].Env()) {
		i += 1
	}
	if i == 0 {
		// failback to prod instances
		j = len(addrs) - 1
		for i < j {
			for isExpect("env", "prod", addrs[i].Env()) && i < j {
				i++
			}
			for !isExpect("env", "prod", addrs[j].Env()) && i < j {
				j--
			}
			addrs[i], addrs[j] = addrs[j], addrs[i]
		}

		if isExpect("env", "prod", addrs[i].Env()) {
			i += 1
		}
	}
	addrs = addrs[:i]

	if len(addrs) == 0 {
		return nil, Err("No hosts left for cluster:%s, env:%s", targetCluster, upstreamEnv)
	}

	ins := make([]kitutil.Instance, 0, len(addrs))
	for _, addr := range addrs {
		ins = append(ins, addr)
	}
	return ins, nil
}
