package kitc

import (
	"context"
	"time"

	"code.byted.org/gopkg/asyncache"
	etcdutil "code.byted.org/gopkg/etcd_util"
	etcdclient "code.byted.org/gopkg/etcd_util/client"
	"code.byted.org/gopkg/logs"
)

var (
	etcdCache  KVStorage
	etcdClient *etcdutil.Client
)

type KVStorage interface {
	Get(key string) (string, error)
	GetOrSet(key string, val string) (string, error)
}

func init() {
	etcdCache = NewEtcdCache()
}

type EtcdStorage struct {
	cache *asyncache.SingleAsyncCache
}

func (es *EtcdStorage) Get(key string) (string, error) {
	val, err := es.cache.Get(key)
	if err != nil {
		return "", err
	}
	ret, _ := val.(string)
	return ret, nil
}

func (es *EtcdStorage) GetOrSet(key, val string) (string, error) {
	v, err := es.cache.Get(key)
	if err != nil && etcdclient.IsKeyNotFound(err) {
		cli, err := etcdutil.GetDefaultClient()
		if err == nil {
			_, err = cli.Create(context.Background(), key, val)
		}
		if err != nil {
			logs.Infof("key=%s err=%s", key, err)
		}
		return val, nil
	}

	if err != nil && err == asyncache.EmptyErr {
		logs.Warnf("key=%s please check this key in etcd, it will use %s as default", key, val)
		return val, nil
	}

	if err != nil {
		logs.Warnf("etcd err, key=%s err=%s, default val %s will be used", key, err, val)
		return val, err
	}

	ret, _ := v.(string)
	return ret, nil
}

// NewEtcdCache wrap etcd client with asyncCache
func NewEtcdCache() KVStorage {
	f := func(key string) (interface{}, error) {
		cli, err := etcdutil.GetDefaultClient()
		if err != nil {
			return nil, err
		}
		val, err := cli.Get(context.Background(), key, nil)
		if err != nil {
			return nil, err
		}
		if val.Node.Value == "" {
			return nil, Err("Get %s is empty", key)
		}
		return val.Node.Value, nil
	}
	return &EtcdStorage{cache: asyncache.NewBlockedAsyncCache(f, time.Second*30)}
}
