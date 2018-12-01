package kitc

import (
	"code.byted.org/kite/kitware"
)

type acler struct {
	storage KVStorage
}

func NewAcler(storage KVStorage) *acler {
	return &acler{storage: storage}
}

func (ac *acler) GetByKey(key string) (string, error) {
	val, err := ac.storage.GetOrSet(key, kitware.ACL_ALLOW)
	if err != nil {
		return "", err
	}
	return val, nil
}
