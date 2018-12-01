package etcdutil

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"path"
	"strings"

	etcd "code.byted.org/gopkg/etcd_util/client"
	"code.byted.org/gopkg/etcd_util/pkg/dns"
	"code.byted.org/gopkg/metrics"
)

type Client struct {
	etcd.KeysAPI // etcd KeysAPI
}

func readSsconfAddrs(fn string) ([]string, error) {
	b, err := ioutil.ReadFile(fn)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(b), "\n")
	for _, l := range lines {
		s := strings.TrimPrefix(l, "etcd_host_port")
		if s != l {
			return strings.Split(s, ","), nil
		}
	}
	return nil, nil
}

// NewClient create client from ss_conf
func NewClient() (*Client, error) {
	// Parse endpoints
	endpoints, err := readSsconfAddrs("/etc/ss_conf/etcd.conf")
	if err == nil && len(endpoints) > 0 {
		for i, e := range endpoints {
			endpoints[i] = "http://" + strings.TrimSpace(e)
		}
	} else {
		endpoints, err = dns.LookupIPAddr("etcdproxy.byted.org", 12*time.Hour)
		for i, e := range endpoints {
			endpoints[i] = "http://" + strings.TrimSpace(e) + ":3379"
		}
	}
	if err != nil {
		return nil, err
	}
	log.Println("etcd endpoints", endpoints)
	return NewClientWithEndpoints(endpoints)
}

// NewClientWithEndpoints create client from endpoints
func NewClientWithEndpoints(endpoints []string) (c *Client, err error) {
	// Create etcd client
	etcdClient, err := etcd.New(etcd.Config{
		Endpoints:               endpoints,
		HeaderTimeoutPerRequest: requestTimeout,
	})
	if err != nil {
		return nil, err
	}

	// Create Client
	c = &Client{
		etcd.NewKeysAPI(etcdClient),
	}
	return c, nil
}

// Get retrieves a set of Nodes from etcd
func (c *Client) Get(ctx context.Context, key string, opts *etcd.GetOptions) (resp *etcd.Response, err error) {
	startTime := time.Now()
	defer func() {
		emitMetrics("get", startTime, err)
	}()

	if opts != nil && !readable(key, opts.Recursive) {
		return nil, fmt.Errorf("Key %s is not recursively readable.", key)
	}

	return c.KeysAPI.Get(ctx, key, opts)
}

// Bellow methods will proxy calls to original implementation, only add metrics to the calls

// Set assigns a new value to a Node identified by a given key. The caller
// may define a set of conditions in the SetOptions. If SetOptions.Dir=true
// than value is ignored.
func (c *Client) Set(ctx context.Context, key, value string, opts *etcd.SetOptions) (resp *etcd.Response, err error) {
	startTime := time.Now()
	defer func() {
		emitMetrics("set", startTime, err)
	}()

	if !updatable(key) {
		return nil, fmt.Errorf("Key %s is not setable.", key)
	}

	return c.KeysAPI.Set(ctx, key, value, opts)
}

// Delete removes a Node identified by the given key, optionally destroying
// all of its children as well. The caller may define a set of required
// conditions in an DeleteOptions object.
func (c *Client) Delete(ctx context.Context, key string, opts *etcd.DeleteOptions) (resp *etcd.Response, err error) {
	startTime := time.Now()
	defer func() {
		emitMetrics("delete", startTime, err)
	}()

	if !updatable(key) {
		return nil, fmt.Errorf("Key %s is not deletable.", key)
	}

	return c.KeysAPI.Delete(ctx, key, opts)
}

// Create is an alias for Set w/ PrevExist=false
func (c *Client) Create(ctx context.Context, key, value string) (resp *etcd.Response, err error) {
	startTime := time.Now()
	defer func() {
		emitMetrics("create", startTime, err)
	}()

	return c.KeysAPI.Create(ctx, key, value)
}

// CreateInOrder is used to atomically create in-order keys within the given directory.
func (c *Client) CreateInOrder(ctx context.Context, dir, value string, opts *etcd.CreateInOrderOptions) (resp *etcd.Response, err error) {
	startTime := time.Now()
	defer func() {
		emitMetrics("createInOrder", startTime, err)
	}()

	return c.KeysAPI.CreateInOrder(ctx, dir, value, opts)
}

// Update is an alias for Set w/ PrevExist=true
func (c *Client) Update(ctx context.Context, key, value string) (resp *etcd.Response, err error) {
	startTime := time.Now()
	defer func() {
		emitMetrics("update", startTime, err)
	}()

	if !updatable(key) {
		return nil, fmt.Errorf("Key %s is not updatable.", key)
	}

	return c.KeysAPI.Update(ctx, key, value)
}

// Watcher builds a new Watcher targeted at a specific Node identified
// by the given key. The Watcher may be configured at creation time
// through a WatcherOptions object. The returned Watcher is designed
// to emit events that happen to a Node, and optionally to its children.
func (c *Client) Watcher(key string, opts *etcd.WatcherOptions) etcd.Watcher {
	startTime := time.Now()
	defer func() {
		emitMetrics("watcher", startTime, nil)
	}()

	return c.KeysAPI.Watcher(key, opts)
}

// Helper methods

const metricsPrefix = "etcd.req"

var langTag = map[string]string{"lang": "go"}
var metricsClient = metrics.NewDefaultMetricsClient(metricsPrefix, true)

// emitMetrics upload metrics to server
func emitMetrics(methodName string, startTime time.Time, err error) {
	if err != nil {
		metricsClient.EmitCounter(methodName+".error.count", 1, metricsPrefix, langTag)
	}
	metricsClient.EmitCounter(methodName+".count", 1, metricsPrefix, langTag)
	metricsClient.EmitTimer(methodName+".latency", toMillisecond(time.Now())-toMillisecond(startTime), metricsPrefix, langTag)
}

// toMillisecond convert time to millisecond
// http://stackoverflow.com/a/24122933/1203241
func toMillisecond(input time.Time) int64 {
	return input.UnixNano() / int64(time.Millisecond)
}

// updatable checks if the given key can be deleted
// It returns true if the key path not less than 2 levels deep
// e.g. `/`, `/web` will be NOT updatable, `/web/ad`, `/web/ad/essay_feed` will be updatable
func updatable(key string) bool {
	return len(strings.Split(strings.Trim(path.Clean(key), "/"), "/")) > 1
}

// readable checks if the given key can be recursively read.
// It returns true if not recursive read or key path not less than 2 levels deep
func readable(key string, recursive bool) bool {
	return !recursive || len(strings.Split(strings.Trim(path.Clean(key), "/"), "/")) > 1
}
