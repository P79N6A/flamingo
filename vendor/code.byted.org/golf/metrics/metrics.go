// Package metrics provides a goroutine safe metrics client
// It provides functions to define and emit counters, timers and stores. It also allow to emit metrics without checking definition if IgnoreMetricsCheck is enabled
package metrics

import (
	"bytes"
	"code.byted.org/golf/buffer_pool"
	"errors"
	"github.com/ugorji/go/codec"
	"net"
	"sort"
	"strings"
)

const (
	MetricsTypeCounter = iota
	MetricsTypeTimer
	MetricsTypeStore
)

var (
	NamespacePrefix           string
	AllMetrics                map[string]int
	AllTags                   map[string]map[string]bool
	IgnoreMetricsCheck        bool
	DuplicatedMetricsErr      = errors.New("duplicated metrics name")
	EmitUndefinedMetricsErr   = errors.New("emit undefined metrics")
	EmitUndefinedTagKErr      = errors.New("emit undefined tagk")
	EmitUndefinedTagVErr      = errors.New("emit undefined tagv")
	ConnectToMetricsServerErr = errors.New("failed to connect to metrics server")
	conn                      *net.UDPConn
)

func Init(server, namespacePrefix string, ignoreMetricsCheck bool) error {
	NamespacePrefix = namespacePrefix
	IgnoreMetricsCheck = ignoreMetricsCheck
	AllMetrics = make(map[string]int)
	AllTags = make(map[string]map[string]bool)
	addr, err := net.ResolveUDPAddr("udp", server)
	if err != nil {
		return err
	}
	conn, err = net.DialUDP("udp", nil, addr)
	if err != nil {
		return err
	}
	return nil
}

func InitWithDefaultServer(namespacePrefix string, ignoreMetricsCheck bool) error {
	return Init("127.0.0.1:9123", namespacePrefix, ignoreMetricsCheck)
}

func IsInitialized() bool {
	return conn != nil
}

func DefineCounter(name, prefix string) error {
	return defineMetrics(name, prefix, MetricsTypeCounter)
}

func DefineTimer(name, prefix string) error {
	return defineMetrics(name, prefix, MetricsTypeTimer)
}

func DefineStore(name, prefix string) error {
	return defineMetrics(name, prefix, MetricsTypeStore)
}

func defineMetrics(name, prefix string, metricsType int) error {
	if IgnoreMetricsCheck {
		return nil
	}
	fullName := c(name, prefix)
	if v, ok := AllMetrics[fullName]; !ok {
		AllMetrics[fullName] = metricsType
	} else if v != metricsType {
		return DuplicatedMetricsErr
	}
	return nil
}

func DefineTagkv(tagk string, tagvs []string) {
	if IgnoreMetricsCheck {
		return
	}
	v, ok := AllTags[tagk]
	if !ok {
		v = make(map[string]bool)
		AllTags[tagk] = v
	}
	for _, tagv := range tagvs {
		v[tagv] = true
	}
}

//
func EmitCounter(name, value, prefix string, tagkv map[string]string) error {
	return emit("counter", name, value, prefix, tagkv)
}
func EmitTimer(name, value, prefix string, tagkv map[string]string) error {
	return emit("timer", name, value, prefix, tagkv)
}
func EmitStore(name, value, prefix string, tagkv map[string]string) error {
	return emit("store", name, value, prefix, tagkv)
}

func tagkv2str(tagkv map[string]string) string {
	if tagkv == nil {
		return ""
	}
	var items []string
	for k, v := range tagkv {
		items = append(items, k+"="+v)
	}
	sort.Strings(items)
	return strings.Join(items, "|")
}

func emitWithCheck(metricsType, name, value, prefix string, tagkv map[string]string, check bool) error {
	if conn == nil {
		return ConnectToMetricsServerErr
	}
	fullName := c(name, prefix)
	if check {
		if _, ok := AllMetrics[fullName]; !ok {
			return EmitUndefinedMetricsErr
		}
		for k, v := range tagkv {
			if tagvs, ok := AllTags[k]; !ok {
				return EmitUndefinedTagKErr
			} else if !tagvs[v] {
				return EmitUndefinedTagVErr
			}
		}
	}
	tagStr := tagkv2str(tagkv)
	req := []string{"emit", metricsType, fullName, value, tagStr, ""}
	buf := buffer_pool.Get()
	defer buffer_pool.Put(buf)
	err := packMsg(buf, req)
	if err != nil {
		return err
	}
	_, err = conn.Write(buf.Bytes())
	return err
}

func emit(metricsType, name, value, prefix string, tagkv map[string]string) error {
	return emitWithCheck(metricsType, name, value, prefix, tagkv, !IgnoreMetricsCheck)
}

func packMsg(buf *bytes.Buffer, req []string) error {
	buf.Reset()
	handler := &codec.MsgpackHandle{}
	encoder := codec.NewEncoder(buf, handler)
	return encoder.Encode(req)
}

// prepend \name with:
// \prefix if prefix is not empty
// or else \namespacePrefix if namespacePrefix is not empty
func c(name, prefix string) string {
	if len(prefix) == 0 {
		if len(NamespacePrefix) == 0 {
			return name
		} else {
			return NamespacePrefix + "." + name
		}
	} else {
		return prefix + "." + name
	}
}
