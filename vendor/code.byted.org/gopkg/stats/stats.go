/*
Example:

if err := stats.DoReport("example"); err != nil{
    fmt.Fprintf(os.Stderr, "DoReport error: %s\n", err)
    os.Exit(-1)
}
*/
package stats

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"code.byted.org/gopkg/metrics"
)

type StatItem struct {
	runtime.MemStats
	Goroutines int64
	NumGC      int64
	GCPauseUs  uint64
}

func TickStat(d time.Duration) <-chan StatItem {
	ret := make(chan StatItem)
	go func() {
		m0 := ReadMemStats()
		if d < time.Second {
			d = time.Second
		}
		for {
			// use time.After for chan blocking issue
			<-time.After(d)
			m1 := ReadMemStats()
			ret <- StatItem{
				MemStats:   m1,
				Goroutines: int64(runtime.NumGoroutine()),
				NumGC:      int64(m1.NumGC - m0.NumGC),
				GCPauseUs:  GCPauseNs(m1, m0) / 1000,
			}
			m0 = m1
		}
	}()
	return ret
}

func ReadMemStats() runtime.MemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m
}

// GCPauseNs cals max(set(new.PauseNs) - set(old.PauseNs))
func GCPauseNs(new runtime.MemStats, old runtime.MemStats) uint64 {
	if new.NumGC <= old.NumGC {
		return new.PauseNs[(new.NumGC+255)%256]
	}
	n := new.NumGC - old.NumGC
	if n > 256 {
		n = 256
	}
	// max PauseNs since last GC
	var maxPauseNs uint64
	for i := uint32(0); i < n; i++ {
		if pauseNs := new.PauseNs[(new.NumGC-i+255)%256]; pauseNs > maxPauseNs {
			maxPauseNs = pauseNs
		}
	}
	return maxPauseNs
}

// GetField is loanutil for emit metrics
func (e StatItem) GetField(key string) interface{} {
	switch key {
	case "HeapAlloc":
		return e.HeapAlloc
	case "StackInuse":
		return e.StackInuse
	case "NumGC":
		return e.NumGC
	case "Goroutines":
		return e.Goroutines
	case "TotalAlloc":
		return e.TotalAlloc
	case "Mallocs":
		return e.Mallocs
	case "Frees":
		return e.Frees
	case "HeapObjects":
		return e.HeapObjects
	case "GCCPUFraction":
		return e.GCCPUFraction
	case "GCPauseUs":
		return e.GCPauseUs
	}
	return nil
}

var (
	MetricsPrefix = "go"
)

type Reporter struct {
	Mcli *metrics.MetricsClientV2

	storeMetrics map[string]string
	timerMetrics map[string]string

	cgroupDirs map[string]string
}

func DoReport(name string) error {
	r := &Reporter{}
	r.Mcli = metrics.NewDefaultMetricsClientV2(MetricsPrefix+"."+name, true)
	if err := r.Init(); err != nil {
		return err
	}
	go r.Reporting()
	return nil
}

func (r *Reporter) Init() error {
	r.storeMetrics = map[string]string{
		"heap.byte":           "HeapAlloc",
		"stack.byte":          "StackInuse",
		"numGcs":              "NumGC",
		"numGos":              "Goroutines",
		"malloc":              "Mallocs",
		"free":                "Frees",
		"totalAllocated.byte": "TotalAlloc",
		"objects":             "HeapObjects",
	}
	r.timerMetrics = map[string]string{
		"gcPause.us": "GCPauseUs",
		"gcCPU":      "GCCPUFraction",
	}
	var s StatItem
	for _, f := range r.storeMetrics {
		if s.GetField(f) == nil {
			return errors.New("[BUG] field " + f + " not found")
		}
	}
	for _, f := range r.timerMetrics {
		if s.GetField(f) == nil {
			return errors.New("[BUG] field " + f + " not found")
		}
	}
	r.cgroupDirs = make(map[string]string)
	data, err := ioutil.ReadFile("/proc/self/cgroup")
	if err == nil {
		// parse
		for _, s := range strings.Split(string(data), "\n") {
			ss := strings.Split(s, ":")
			if len(ss) != 3 {
				continue
			}
			r.cgroupDirs[ss[1]] = filepath.Join("/sys/fs/cgroup/", ss[1], strings.TrimSpace(ss[2]))
		}
	}
	return nil
}

func (r *Reporter) Reporting() {
	for e := range TickStat(time.Second) {
		for k, f := range r.storeMetrics {
			r.Mcli.EmitStore(k, e.GetField(f))
		}
		for k, f := range r.timerMetrics {
			r.Mcli.EmitTimer(k, e.GetField(f))
		}
		r.reportCPUStat()
		r.reportCPUUsage()
	}
}

func (r *Reporter) reportCPUStat() {
	dir := r.cgroupDirs["cpu,cpuacct"]
	if dir == "" {
		return
	}
	fn := filepath.Join(dir, "cpu.stat")
	data, err := ioutil.ReadFile(fn)
	if err != nil {
		return
	}
	var nr_periods, nr_throttled, throttled_time int64
	n, _ := fmt.Sscanf(string(data),
		"nr_periods %d\nnr_throttled %d\nthrottled_time %d",
		&nr_periods, &nr_throttled, &throttled_time)
	if n != 3 {
		return
	}
	r.Mcli.EmitStore("cgroup.nr_periods", nr_periods)
	r.Mcli.EmitStore("cgroup.nr_throttled", nr_throttled)
	r.Mcli.EmitStore("cgroup.throttled_time", throttled_time)
}

func (r *Reporter) reportCPUUsage() {
	dir := r.cgroupDirs["cpu,cpuacct"]
	if dir == "" {
		return
	}
	fn := filepath.Join(dir, "cpuacct.usage")
	data, err := ioutil.ReadFile(fn)
	if err != nil {
		return
	}
	var cpuacct_usage int64
	n, _ := fmt.Sscanf(string(data), "%d", &cpuacct_usage)
	if n != 1 {
		return
	}
	r.Mcli.EmitStore("cgroup.cpuacct.usage", cpuacct_usage)
}
