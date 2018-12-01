package kitc

import (
	"sort"
	"sync"
	"time"

	"code.byted.org/gopkg/asyncache"
	"code.byted.org/gopkg/logs"
)

// CacheService package an ikService with async cache
type CacheService struct {
	ikService IKService
	cache     *asyncache.SingleAsyncCache
}

// NewCacheService return an iService which async cache
func NewCacheService(iks IKService) *CacheService {
	insCache := make(map[string][]*Instance) // used to trace the change of instances in f
	// lock protect for insCache
	var lock sync.Mutex
	// this func will run in asyncache,
	// key is idc/cluster/env
	f := func(key string) (interface{}, error) {
		instances := iks.Lookup(key)
		if len(instances) == 0 {
			return nil, Err("No instances are discoverd by service=%s idc=%s", iks.Name(), key)
		}

		lock.Lock()
		old := insCache[key]
		lock.Unlock()

		if isSameInsList(instances, old) == false {
			logs.Warnf("service=%s dc=%s old=%v new=%v address have changed",
				iks.Name(), key, insList2Addrs(old), insList2Addrs(instances))
			lock.Lock()
			insCache[key] = instances
			lock.Unlock()
		}

		return instances, nil
	}

	cache := asyncache.NewBlockedAsyncCache(f, time.Second*30)
	return &CacheService{
		ikService: iks,
		cache:     cache,
	}
}

func (cs *CacheService) Lookup(idc string) []*Instance {
	v, err := cs.cache.Get(idc)
	if err != nil {
		logs.Error("serviceName=%s idc=%s err=%s", cs.ikService.Name(), idc, err)
		return nil
	}
	return v.([]*Instance)
}

func (cs *CacheService) Name() string {
	return cs.ikService.Name()
}

// isSameInsList compares the "host:port"" of all instances in insList0 and insList1 to
// check whether insList0 and insList1 are same.
func isSameInsList(insList0, insList1 []*Instance) bool {
	if insList0 == nil && insList1 == nil {
		return true
	} else if insList0 == nil || insList1 == nil {
		return false
	} else if len(insList0) != len(insList1) {
		return false
	}

	/*
		these two string slices wouldn't cause performance problem,
		because isSameInsList() will only called by asyncache,
		which call it per 5~10s.
	*/
	addrs0 := insList2Addrs(insList0)
	addrs1 := insList2Addrs(insList1)
	sort.Strings(addrs0)
	sort.Strings(addrs1)

	for i := 0; i < len(addrs0); i++ {
		if addrs0[i] != addrs1[i] {
			return false
		}
	}

	return true
}

// insList2Addrs converts this insList to a string slice which consist of all "host:port" in these instances
func insList2Addrs(insList []*Instance) []string {
	addrs := make([]string, 0, len(insList))
	for _, ins := range insList {
		addrs = append(addrs, ins.Host()+":"+ins.Port())
	}
	return addrs
}
