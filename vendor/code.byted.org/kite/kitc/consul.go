package kitc

import (
	"os"
	"strconv"
	"strings"

	"code.byted.org/golf/consul"
	"code.byted.org/gopkg/env"
	"code.byted.org/gopkg/logs"
)

const (
	DefaultIDC = "default"
)

var (
	consulAgentHost string = "127.0.0.1"
	consulAgentPort int    = 2280
)

func init() {
	if host := os.Getenv("TCE_HOST_IP"); host != "" {
		consulAgentHost = host
	}
}

// LocalIDC return idc's name of current service
// first read env val RUNTIME_IDC_NAME
func LocalIDC() string {
	idc := env.IDC()
	if idc == env.UnknownIDC {
		return DefaultIDC
	}
	return idc
}

type ConsulService struct {
	name string
}

func NewConsulService(name string) IKService {
	return &ConsulService{name}
}

func (cs *ConsulService) Name() string {
	return cs.name
}

// Lookup return a list of instances
func (cs *ConsulService) Lookup(idc string) []*Instance {
	idc = strings.TrimSpace(idc)
	items, err := consul.TranslateOneOnHost(cs.name+".service."+idc, consulAgentHost, consulAgentPort)
	if err != nil {
		logs.Error("consul.TranslateOne error: %s", err)
		return nil
	}

	var ret []*Instance
	for _, ins := range items {
		ret = append(ret, NewInstance(ins.Host, strconv.Itoa(ins.Port), ins.Tags))
	}
	return ret
}

// ClusterFilter
func ClusterFilter(cluster string, data map[string]string) bool {
	if cluster == "" {
		if val, ok := data[ClusterKey]; !ok || val == "default" {
			return true
		}
		return false
	}
	if strEqual(cluster, data[ClusterKey]) {
		return true
	}
	return false
}

// EnvFilter
func EnvFilter(env string, data map[string]string) bool {
	if env == "" {
		if val, ok := data[EnvKey]; !ok || val == "default" {
			return true
		}
		return false
	}
	if strEqual(env, data[EnvKey]) {
		return true
	}
	return false
}

func strEqual(str1, str2 string) bool {
	if strings.TrimSpace(str1) == strings.TrimSpace(str2) {
		return true
	}
	return false
}
