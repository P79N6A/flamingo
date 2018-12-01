package etcdutil

import (
	"context"
	"fmt"
	"sync"
	"time"

	etcd "code.byted.org/gopkg/etcd_util/client"
	"code.byted.org/gopkg/ttlcache"
)

// requestTimeout is the default value for query etcd server
var requestTimeout = 100 * time.Millisecond

const defaultExpiration = 1 * time.Minute

// cache will be shared for all keys
var cache *ttlcache.Cache = ttlcache.NewCache(defaultExpiration)

// defaultClient is created for easy use
var defaultClient *Client
var defaultClientMu sync.RWMutex

func GetDefaultClient() (*Client, error) {
	defaultClientMu.RLock()
	cli := defaultClient
	defaultClientMu.RUnlock()
	if cli != nil {
		return cli, nil
	}

	defaultClientMu.Lock()
	defer defaultClientMu.Unlock()
	cli = defaultClient
	if cli != nil {
		return cli, nil
	}
	var err error
	defaultClient, err = NewClient()
	return defaultClient, err
}

func SetRequestTimeout(timeout time.Duration) {
	requestTimeout = timeout
}

// Get using DefaultClient
func Get(key string, defaultValue string) (string, error) {
	return GetWithOptions(key, defaultValue, nil)
}

// GetWithOptions using DefaultClient
func GetWithOptions(key string, defaultValue string, opts *etcd.GetOptions) (string, error) {
	cli, err := GetDefaultClient()
	if err != nil {
		return defaultValue, err
	}
	return cli.GetWithCacheAndOpts(key, defaultValue, opts)
}

// Set using DefaultClient
func Set(key string, value string) (string, error) {
	return SetWithOptions(key, value, nil)
}

// SetWithOptions using DefaultClient
func SetWithOptions(key string, value string, opts *etcd.SetOptions) (string, error) {
	cli, err := GetDefaultClient()
	if err != nil {
		return "", err
	}
	resp, err := cli.Set(context.TODO(), key, value, opts)
	if err != nil || resp == nil {
		return "", err
	}
	cache.Set(key, value)
	return resp.Node.Value, nil
}

func (c *Client) GetWithCache(key string, defaultValue string) (string, error) {
	return c.GetWithCacheAndOpts(key, defaultValue, nil)
}

func (c *Client) GetWithCacheAndOpts(key string, defaultValue string, opts *etcd.GetOptions) (string, error) {
	if len(key) == 0 {
		return defaultValue, fmt.Errorf("Can't get from empty key")
	}

	// Get from cache
	value, exists := cache.Get(key)
	if exists {
		// key in cache
		return value, nil
	}
	value = defaultValue

	// key not in cache or expired
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	resp, err := c.Get(ctx, key, opts)
	if etcd.IsKeyNotFound(err) {
		err = nil
	} else if resp != nil && resp.Node != nil && len(resp.Node.Value) != 0 {
		value = resp.Node.Value
	}
	cache.Set(key, value)
	return value, err
}
