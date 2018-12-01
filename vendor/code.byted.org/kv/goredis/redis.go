package goredis

import (
	"math/rand"
	"time"

	"code.byted.org/kv/redis-v6"

	"code.byted.org/gopkg/logs"
)

type Client struct {
	*redis.Client

	cluster            string
	psm                string
	metricsServiceName string
}

// NewClient will create a new client with cluster name use the default timeout settings
func NewClient(cluster string) (*Client, error) {
	opt := NewOption()
	return NewClientWithOption(cluster, opt)
}

// NewClientWithOption will use user specified timeout settings in option
func NewClientWithOption(cluster string, opt *Option) (*Client, error) {
	servers, err := loadConfByClusterName(cluster, opt.configFilePath, opt.useConsul)
	if err != nil {
		return nil, err
	}
	logs.Info("Cluster %v's server list is %v", cluster, servers)
	return NewClientWithServers(cluster, servers, opt)
}

// NewClientWithServers will create a new client with specified servers and timeout in option
func NewClientWithServers(cluster string, servers []string, opt *Option) (*Client, error) {
	if len(servers) == 0 {
		return nil, ErrEmptyServerList
	}
	serversCh := make(chan []string, 1)
	serversCh <- servers
	opt.Addr = servers[rand.Intn(len(servers))]

	dial := NewDialer(servers, serversCh, opt)
	opt.Dialer = dial.getDialConn
	cli := &Client{
		Client:             redis.NewClient(opt.Options),
		cluster:            GetClusterName(cluster),
		psm:                checkPsm(),
		metricsServiceName: GetPSMClusterName(cluster),
	}
	cli.WrapProcess(cli.metricsWrapProcess)

	if opt.autoLoadConf {
		autoLoadConf(cli.cluster, serversCh, opt)
	}
	// go func() {
	// 	for {
	// 		logs.Info("Redis poolstats: %v", *cli.PoolStats())
	// 		time.Sleep(time.Second)
	// 	}
	// }()
	return cli, nil
}

func (c *Client) metricsWrapProcess(oldProcess func(cmd redis.Cmder) error) func(cmd redis.Cmder) error {
	return func(cmd redis.Cmder) error {
		cmdStr := cmd.Name()

		start := time.Now().UnixNano()
		var err error
		// var reqCmdSize int = bytes.Count([]byte(cmd.String()), nil) - 1
		// var respCmdSize int
		defer func() {
			var callStatus string
			if err == redis.Nil {
				callStatus = CALLSTATUS_MISS
			} else if err != nil {
				callStatus = CALLSTATUS_ERROR
			} else {
				callStatus = CALLSTATUS_SUCCESS
			}
			addCallMetrics(
				cmdStr,
				(time.Now().UnixNano()-start)/1000,
				callStatus,
				c.cluster,
				c.psm,
				c.metricsServiceName)
		}()

		err = oldProcess(cmd)
		// respCmdSize = bytes.Count([]byte(cmd.String()), nil) - 1
		return err
	}
}
func (c *Client) Pipeline() *Pipeline {
	pipe := &Pipeline{
		c.Client.Pipeline(),
		c.cluster,
		c.psm,
		c.metricsServiceName,
		"pipeline", // default pipeline name
	}
	return pipe
}

// this func will create a pipeline with name user specified
// the name will be used for pipeline metrics
func (c *Client) NewPipeline(pipelineName string) *Pipeline {
	pipe := &Pipeline{
		c.Client.Pipeline(),
		c.cluster,
		c.psm,
		c.metricsServiceName,
		pipelineName,
	}
	return pipe
}
