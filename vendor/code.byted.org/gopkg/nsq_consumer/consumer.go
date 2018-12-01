package consumer

import (
	"errors"
	"log"
	"net/url"
	"strings"
	"time"

	"code.byted.org/gopkg/metrics"
	"github.com/bitly/go-nsq"
)

type ConsumerCtx struct {
	Topic         string
	Channel       string
	LookupdStr    string
	LookupdAddrs  []string
	Consumer      *nsq.Consumer
	MsgHandler    Handler
	Concurrency   int
	MetricsClient *metrics.MetricsClient

	ThroughputKey      string
	ErrorThroughputKey string
	LatencyKey         string
	Tagkv              map[string]string
}

const (
	MetricsPrefix         = "nsq.consumer"
	ThroughputSuffix      = ".throughput"
	ErrorThroughputSuffix = "error.throughput"
	LatencySuffix         = ".latency"
	DefaultUA             = "go-nsq consumer framwork 0.0.1-alpha"
)

type Handler interface {
	HandleMessage(msg []byte) error
}

func AddChannelLookupdAddrs(lookupdAddrs []string, channel string) []string {
	xlookupds := []string{}
	for _, endPoint := range lookupdAddrs {
		u, err := url.Parse(endPoint)
		if err != nil {
			log.Fatal("Invalid lookupd addrs: %v", err)
		}
		q := u.Query()
		q.Set("channel", channel)
		u.RawQuery = q.Encode()
		//log.Println("The translated URL: ", u.String())
		xlookupds = append(xlookupds, u.String())
	}
	return xlookupds
}

func (ctx *ConsumerCtx) HandleMessage(m *nsq.Message) error {
	//First accumulate the metrics
	st := time.Now().UnixNano()
	//Second Call the really msg handler
	err := ctx.MsgHandler.HandleMessage(m.Body)
	et := time.Now().UnixNano()

	//Translate to time.Millisecond
	duration := (st - et) / (int64)(time.Millisecond)

	ctx.MetricsClient.EmitCounter(ctx.ThroughputKey, 1, MetricsPrefix, ctx.Tagkv)
	ctx.MetricsClient.EmitTimer(ctx.LatencyKey, duration, MetricsPrefix, ctx.Tagkv)

	if err != nil {
		ctx.MetricsClient.EmitCounter(ctx.ErrorThroughputKey, 1, MetricsPrefix, ctx.Tagkv)
	}
	return err
}

func InitConsumer(topic, channel, lookupdStr string, handler Handler, concurrency int) (error, *ConsumerCtx) {
	if concurrency <= 0 {
		return errors.New("concurrency Must bigger than 0"), nil
	}

	//Split the lookupds string, default delimiters ','
	lookupds := strings.Split(lookupdStr, ",")
	if len(lookupds) <= 0 {
		return errors.New("Lookupd endPoint is empty"), nil
	}

	lookupds = AddChannelLookupdAddrs(lookupds, channel)

	cCfg := nsq.NewConfig()
	cCfg.UserAgent = DefaultUA
	cCfg.MaxInFlight = 100
	cCfg.HeartbeatInterval = 59

	consumer, err := nsq.NewConsumer(topic, channel, cCfg)
	if err != nil {
		return err, nil
	}

	metricsClient := metrics.NewDefaultMetricsClient(MetricsPrefix, false)
	throughputKey := strings.Replace(topic+ThroughputSuffix, "#", "_", -1)
	errorThroughputKey := strings.Replace(topic+ErrorThroughputSuffix, "#", "_", -1)
	latencyKey := strings.Replace(topic+LatencySuffix, "#", "_", -1)

	metricsClient.DefineCounter(throughputKey, MetricsPrefix)
	metricsClient.DefineCounter(errorThroughputKey, MetricsPrefix)
	metricsClient.DefineTimer(latencyKey, MetricsPrefix)

	ctx := &ConsumerCtx{
		Topic:         topic,
		Channel:       channel,
		LookupdStr:    lookupdStr,
		LookupdAddrs:  lookupds,
		Concurrency:   concurrency,
		Consumer:      consumer,
		MsgHandler:    handler,
		MetricsClient: metricsClient,

		ThroughputKey:      throughputKey,
		LatencyKey:         latencyKey,
		Tagkv:              map[string]string{"channel": strings.Replace(channel, "#", "_", -1)},
		ErrorThroughputKey: errorThroughputKey,
	}
	//fmt.Println("hello connec to lookupds", lookupds)
	consumer.AddConcurrentHandlers(ctx, concurrency)

	err = consumer.ConnectToNSQLookupds(lookupds)
	if err != nil {
		return err, nil
	}
	return nil, ctx
}
