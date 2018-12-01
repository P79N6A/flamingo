// 用户行为数据收集客户端
package mario_collector

import (
	"code.byted.org/data/databus_client"
	"code.byted.org/data/mario_collector/pb_event"
	"code.byted.org/data/mario_collector/pb_server_impression"
	"code.byted.org/gopkg/metrics"
	"github.com/golang/protobuf/proto"
	"net"
	"strings"
	"time"
)

const (
	METRICS_PREFIX = "data.mario.collector"
)

type MarioCollector struct {
	databusCollector *databus_client.DatabusCollector
}

var (
	metricsClient             *metrics.MetricsClient
	event_channel             string
	server_impression_channel string
)

func hasIpPrefix(prefix string) bool {
	ifaces, _ := net.Interfaces()
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		for _, addr := range addrs {
			var ip string
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP.String()
			case *net.IPAddr:
				ip = v.IP.String()
			}
			if strings.HasPrefix(ip, prefix) {
				return true
			}
		}
	}
	return false
}

func isI18nVaEnv() bool {
	return hasIpPrefix("10.100")
}

func isI18nSgEnv() bool {
	return hasIpPrefix("10.101")
}

func init() {
	metricsClient = metrics.NewDefaultMetricsClient(METRICS_PREFIX, false)
	metricsClient.DefineCounter("event.success", "")
	metricsClient.DefineCounter("event.fail", "")
	metricsClient.DefineCounter("server_impression.success", "")
	metricsClient.DefineCounter("server_impression.fail", "")

	if isI18nVaEnv() {
		event_channel = "mario_pb_event_i18n"
		server_impression_channel = "mario_pb_impression_i18n"
	} else if isI18nSgEnv() {
		event_channel = "mario_pb_event_i18n_sg"
		server_impression_channel = "mario_pb_impression_i18n_sg"
	} else {
		event_channel = "mario_pb_event"
		server_impression_channel = "mario_pb_impression"
	}
}

//尽量复用MarioCollector对象
func NewMarioCollector() (collector *MarioCollector) {
	collector = &MarioCollector{
		databusCollector: databus_client.NewDefaultCollector(),
	}
	return
}

func (this *MarioCollector) CollectEvents(caller string, user *pb_event.User, header *pb_event.Header, events []*pb_event.Event) error {
	ts := uint32(time.Now().Unix())
	message := &pb_event.MarioEvents{
		Caller:     &caller,
		ServerTime: &ts,
		User:       user,
		Header:     header,
		Events:     events,
	}
	bytes, err := proto.Marshal(message)
	if err != nil {
		return err
	}
	err = this.databusCollector.Collect(event_channel, bytes, []byte(*user.UserUniqueId), 0)
	tagkv := map[string]string{"caller": caller}
	var metric string
	if err == nil {
		metric = "event.success"
	} else {
		metric = "event.fail"
	}
	metricsClient.EmitCounter(metric, 1, "", tagkv)
	return err
}

func (this *MarioCollector) CollectEvent(caller string, user *pb_event.User, header *pb_event.Header, event *pb_event.Event) error {
	return this.CollectEvents(caller, user, header, []*pb_event.Event{event})
}

func (this *MarioCollector) CollectServerImpression(caller string, impression *pb_server_impression.ServerImpression) error {
	ts := uint32(time.Now().Unix())
	impression.Caller = &caller
	impression.ServerTime = &ts
	bytes, err := proto.Marshal(impression)
	if err != nil {
		return err
	}
	err = this.databusCollector.Collect(server_impression_channel, bytes, []byte(*impression.User.UserUniqueId), 0)
	tagkv := map[string]string{"caller": caller}
	var metric string
	if err == nil {
		metric = "server_impression.success"
	} else {
		metric = "server_impression.fail"
	}
	metricsClient.EmitCounter(metric, 1, "", tagkv)
	return err
}

func (this *MarioCollector) Close() error {
	return this.databusCollector.Close()
}
