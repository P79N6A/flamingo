package kite

import (
	"encoding/json"
	"os"
	"time"

	"code.byted.org/gopkg/grass"
	"code.byted.org/gopkg/logs"
)

// Profile is a schema wrap info
type Profile struct {
	Type    string `json:"type"`
	PSM     string `json:"psm"`
	Cluster string `json:"cluster"`
	// RFC3339 format
	Time string `json:"time"`
	Host string `json:"host"`
	Data string `json:"data"`
}

type kiteReporter struct {
	hostname    string
	grassClient *grass.GrassClient
}

func newKiteReporter() *kiteReporter {
	hostname, _ := os.Hostname()
	client, _ := grass.NewGrassClient()
	return &kiteReporter{
		hostname:    hostname,
		grassClient: client,
	}
}

func (cbr *kiteReporter) Report(topic, typ string, data []byte) {
	body := Profile{
		Type:    typ,
		PSM:     ServiceName,
		Cluster: ServiceCluster,
		Time:    utcTime(),
		Host:    cbr.hostname,
	}
	body.Data = string(data)

	data, _ = json.Marshal(body)
	cbr.grassClient.Plant(cbr.hostname, topic, string(data))
}

func utcTime() string {
	now := time.Now()
	utc, _ := time.LoadLocation("")
	now = now.In(utc)
	return now.Format(time.RFC3339)
}

type debugReporter struct{}

func (r *debugReporter) Report(topic string, data []byte) error {
	logs.Infof("topic=%s profile=%s", topic, string(data))
	return nil
}
