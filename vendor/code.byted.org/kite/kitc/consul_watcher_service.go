package kitc

import (
	"bytes"
	"fmt"
	"strconv"
	"sync"
	"time"

	"code.byted.org/golf/consul"
	"code.byted.org/gopkg/logs"
)

const (
	FIRST_WATCH_TIMEOUT_SECONDS = 0
	MIN_WATCH_TIMEOUT_SECONDS   = 5
	WATCH_TIMEOUT_SECONDS       = 180
)

// ConsulWatcherService implements IKService interface, and provide a
// watcher based service discovery mechanism to achieve real-time backend change awareness
type ConsulWatcherService struct {
	IKService

	instances  map[string][]*Instance
	watchers   map[string]*Watcher
	instanceMu sync.RWMutex
	watcherMu  sync.Mutex
}

func NewConsulWatcherService(ikservice IKService) *ConsulWatcherService {
	return &ConsulWatcherService{
		IKService: ikservice,
		instances: make(map[string][]*Instance),
		watchers:  make(map[string]*Watcher),
	}
}

func (s *ConsulWatcherService) Name() string {
	return s.IKService.Name()
}

func (s *ConsulWatcherService) Lookup(idc string) []*Instance {
	s.instanceMu.RLock()
	instances, ok := s.instances[idc]
	s.instanceMu.RUnlock()
	if ok {
		return instances
	}

	s.watcherMu.Lock()
	if _, ok := s.watchers[idc]; !ok {
		watcher := NewWatcher(s.Name(), idc, s.onConfigChanged)
		watcher.Start()
		s.watchers[idc] = watcher
	}
	s.watcherMu.Unlock()

	s.instanceMu.RLock()
	defer s.instanceMu.RUnlock()
	return s.instances[idc]
}

func (s *ConsulWatcherService) onConfigChanged(dc string, endpoints []*consul.Endpoint) {
	var instances []*Instance
	for _, endpoint := range endpoints {
		instances = append(instances, &Instance{
			host: endpoint.Host,
			port: strconv.Itoa(endpoint.Port),
			tags: endpoint.Tags,
		})
	}
	s.instanceMu.Lock()
	defer s.instanceMu.Unlock()
	s.instances[dc] = instances
}

type ConfigChangedCB func(string, []*consul.Endpoint)

// watcher watches service endpoints and return on endpoint list changed or
// watch timeout elapsed
// on successful and endpoint changes, it call ConfigChangedCB to update endpoints
type Watcher struct {
	service         string
	dc              string
	index           uint64
	OnConfigChanged ConfigChangedCB
}

func NewWatcher(service, dc string, cb ConfigChangedCB) *Watcher {
	return &Watcher{
		service:         fmt.Sprintf("%s.service.%s", service, dc),
		dc:              dc,
		index:           0,
		OnConfigChanged: cb,
	}
}

func (w *Watcher) Start() {
	logs.Info("Start watch %s", w.service)
	// 第一次同步获取service endpoint列表
	if err := w.watchOnce(FIRST_WATCH_TIMEOUT_SECONDS); err != nil {
		logs.Error("First watch failed: %s, %v", w.service, err)
	}
	go w.run()
}

func (w *Watcher) run() {
	for {
		ts := time.Now()
		if err := w.watchOnce(WATCH_TIMEOUT_SECONDS); err != nil {
			// 失败时至少MIN_WATCH_TIMEOUT_SECONDS时间间隔才执行一次, 避免agent故障时这里反复循环
			time.Sleep(ts.Add(MIN_WATCH_TIMEOUT_SECONDS * time.Second).Sub(time.Now()))
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (w *Watcher) watchOnce(timeoutSecs int) error {
	logs.Debug("Watch service: %s", w.service)
	endpointsMap, index, err := consul.WatchMultiple([]string{w.service}, w.index, timeoutSecs)
	if err != nil {
		logs.Error("Watch service failed: %s, %v", w.service, err)
		return err
	}
	if index == w.index {
		// index不变表明列表没有变化
		return nil
	}

	w.index = index
	endpoints, ok := endpointsMap[w.service]
	if !ok {
		// should never happen, just for safety
		err := fmt.Errorf("Unexpected error: %s, %v", w.service, endpointsMap)
		logs.Error(err.Error())
		return err
	}
	if w.OnConfigChanged != nil {
		w.OnConfigChanged(w.dc, endpoints)
	}

	// log out new config
	var buf bytes.Buffer
	for _, endpoint := range endpoints {
		if buf.Len() != 0 {
			buf.WriteByte(',')
		}
		buf.WriteString(endpoint.Host)
		buf.WriteByte(':')
		buf.WriteString(strconv.Itoa(endpoint.Port))
	}
	logs.Info("Service %s config changed: %s", w.service, buf.String())
	return nil
}
