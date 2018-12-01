package net2

import (
	"container/list"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"sync"
	"time"
)

type dnsItem struct {
	UpdatedAt time.Time `json:"u"`
	Hosts     []string  `json:"h"`
}

var (
	resolver   = net.Resolver{PreferGo: true}
	dnsTimeout = time.Second

	tmpfsCacheTimeout = 12 * time.Hour

	dnsMu      sync.RWMutex
	dnscache   = make(map[string]dnsItem)
	dnsWaiters = make(map[string]*list.List)
)

var errEmpty = errors.New("returns empty host")

func LookupIPAddr(name string, cachetime time.Duration) ([]string, error) {
	now := time.Now()
	dnsMu.RLock()
	v := dnscache[name]
	dnsMu.RUnlock()

	if len(v.Hosts) > 0 && now.Sub(v.UpdatedAt) < cachetime {
		return v.Hosts, nil
	}
	if len(v.Hosts) == 0 && now.Sub(v.UpdatedAt) < time.Second {
		return nil, errEmpty
	}

	b, err := ioutil.ReadFile(genIPAddrFilename(name))
	if len(b) > 0 {
		if er := json.Unmarshal(b, &v); er != nil {
			log.Println("[net2] json.Unmarshal", err)
		}
	}
	if os.IsNotExist(err) || now.Sub(v.UpdatedAt) > tmpfsCacheTimeout {
		ret, err := lookupIPAddr(name, cachetime)
		if err == nil {
			return ret, nil
		}
		if len(v.Hosts) > 0 {
			log.Println("[net2] lookupIPAddr", err)
			return v.Hosts, nil
		}
		return nil, err
	}
	v.UpdatedAt = now
	dnsMu.Lock()
	dnscache[name] = v
	dnsMu.Unlock()
	return v.Hosts, nil
}

var cu, _ = user.Current()

func genIPAddrFilename(name string) string {
	if cu != nil {
		return filepath.Join(os.TempDir(), name+"-"+cu.Username+".json")
	}
	return filepath.Join(os.TempDir(), name+".json")
}
