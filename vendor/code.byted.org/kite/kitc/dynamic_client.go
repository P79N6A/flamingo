package kitc

import (
	"encoding/json"

	"code.byted.org/kite/kitutil"
	"code.byted.org/kite/kitware"
)

var defaultTPolicyString string

func init() {
	defaultTPolicy := kitware.RPCPolicy{
		RetryTimes:          0,
		ConnectTimeout:      int64(defaultConnTimeout) / 1000000,
		ConnectRetryMaxTime: int64(defaultConnRetryTimeout) / 1000000,
		ReadTimeout:         int64(defaultReadWriteTimeout) / 1000000,
		WriteTimeout:        int64(defaultReadWriteTimeout) / 1000000,
		TrafficPolicy:       []kitutil.TPolicy{},
	}
	data, _ := json.Marshal(defaultTPolicy)
	defaultTPolicyString = string(data)
}

type dynamicClient struct {
	storage KVStorage
}

func NewDynamicClient(storage KVStorage) *dynamicClient {
	return &dynamicClient{storage: storage}
}

func (c *dynamicClient) GetByKey(key string) ([]byte, error) {
	val, err := c.storage.GetOrSet(key, defaultTPolicyString)
	if err != nil {
		return nil, err
	}
	return []byte(val), nil
}
