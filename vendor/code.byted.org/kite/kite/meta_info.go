package kite

import (
	"strings"

	"code.byted.org/gopkg/logs"
	"code.byted.org/gopkg/thrift"
)

var (
	// ServerMetadata holds the server's metaData
	ServerMetadata = serverMetadata{}
)

type thriftConf struct {
	InThriftTransport  string
	InThriftProtocol   string
	OutThriftTransport string
	OutThriftProtocol  string
	ThriftVersion      string
}

type serverMetadata struct {
	PSM              string
	Cluster          string
	Language         string
	Framework        string
	FrameworkVersion string
	Protocol         string
	IP               string
	Port             string
	DebugPort        string
	ThriftConfig     thriftConf
}

func initServerMetadata() {
	ServerMetadata.PSM = ServiceName
	// TODO Cluster will be supported
	ServerMetadata.Cluster = ServiceCluster
	ServerMetadata.Language = "go"
	ServerMetadata.Framework = "kite"
	ServerMetadata.FrameworkVersion = "v2.0"
	ServerMetadata.Protocol = "thrift"
	ServerMetadata.IP = LocalIp
	ServerMetadata.Port = ServicePort
	ServerMetadata.DebugPort = DebugServerPort
	if p := strings.Index(DebugServerPort, ":"); p != -1 {
		ServerMetadata.DebugPort = DebugServerPort[p+1:]
	}
	ServerMetadata.ThriftConfig.ThriftVersion = "0.9.2"
	ServerMetadata.ThriftConfig.InThriftProtocol = getThriftProtocolType(RpcService.protocolFactory)
	ServerMetadata.ThriftConfig.OutThriftProtocol = getThriftProtocolType(RpcService.protocolFactory)
	ServerMetadata.ThriftConfig.InThriftTransport = getThriftTransportType(RpcService.transportFactory)
	ServerMetadata.ThriftConfig.OutThriftTransport = getThriftTransportType(RpcService.transportFactory)
}

// ReportMetadata report server's metadata
func ReportMetadata() {

	initServerMetadata()

	infos := make(map[string]string)
	infos["psm"] = ServerMetadata.PSM
	infos["cluster"] = ServerMetadata.Cluster
	infos["language"] = ServerMetadata.Language
	infos["framework"] = ServerMetadata.Framework
	infos["framework_version"] = ServerMetadata.FrameworkVersion
	infos["protocol"] = ServerMetadata.Protocol
	infos["ip"] = ServerMetadata.IP
	infos["port"] = ServerMetadata.Port
	infos["debug_port"] = ServerMetadata.DebugPort
	infos["thrift_in_protocol"] = ServerMetadata.ThriftConfig.InThriftProtocol
	infos["thrift_in_transport"] = ServerMetadata.ThriftConfig.InThriftTransport
	infos["thrift_out_protocol"] = ServerMetadata.ThriftConfig.OutThriftProtocol
	infos["thrift_out_transport"] = ServerMetadata.ThriftConfig.OutThriftTransport
	infos["thrift_version"] = ServerMetadata.ThriftConfig.ThriftVersion

	defer func() {
		if r := recover(); r != nil {
			logs.Warn("Report server's metadata unsuccessfully, but it can be ignored: %v", r)
		}
	}()

	if err := BagentClient.ReportInfo(infos); err != nil {
		logs.Warn("Report server's metadata unsuccessfully, bu it can be ignored: %s", err)
	}
}

func getThriftProtocolType(protocolFactory thrift.TProtocolFactory) string {
	switch protocolFactory.(type) {
	case *thrift.TBinaryProtocolFactory:
		return "binary"
	case *thrift.TCompactProtocolFactory:
		return "compact"
	}

	return "other"
}

func getThriftTransportType(transportFactory thrift.TTransportFactory) string {
	switch transportFactory.(type) {
	case *thrift.TBufferedTransportFactory:
		return "buffered"
	}

	return "other"
}
