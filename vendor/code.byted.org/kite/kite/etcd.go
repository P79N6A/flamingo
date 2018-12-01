package kite

import (
	"code.byted.org/kite/kitc"
)

var etcdCache kitc.KVStorage

func init() {
	etcdCache = kitc.NewEtcdCache()
}
