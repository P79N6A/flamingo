// +build !go1.8

package net2

import (
	"container/list"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"
)

func lookupIPAddr(name string, cachetime time.Duration) ([]string, error) {
	ctx, _ := context.WithDeadline(context.TODO(), time.Now().Add(dnsTimeout))
	dnsMu.Lock()
	waiters := dnsWaiters[name]
	if waiters != nil {
		ready := make(chan struct{})
		waiters.PushBack(ready)
		dnsMu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ready:
		}
		return LookupIPAddr(name, cachetime)
	}
	dnsWaiters[name] = list.New()
	dnsMu.Unlock()
	defer func() {
		dnsMu.Lock()
		ws := dnsWaiters[name]
		delete(dnsWaiters, name)
		dnsMu.Unlock()
		for e := ws.Front(); e != nil; e = e.Next() {
			ch := e.Value.(chan struct{})
			close(ch)
		}
	}()
	ret, err := net.LookupHost(name)
	if err != nil {
		return nil, err
	}
	item := dnsItem{UpdatedAt: time.Now(), Hosts: ret}
	dnsMu.Lock()
	if len(item.Hosts) > 0 || len(dnscache[name].Hosts) == 0 {
		dnscache[name] = item
	}
	dnsMu.Unlock()
	if len(ret) == 0 {
		return nil, errEmpty
	}
	// file cache
	f, err := ioutil.TempFile("", name)
	if err != nil {
		log.Println("[net2] LookupIPAddr: open temp file err:", err)
		return ret, nil
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	if err := enc.Encode(item); err != nil {
		log.Println("[net2] LookupIPAddr: write temp file err:", err)
		return ret, nil
	}
	os.Rename(f.Name(), genIPAddrFilename(name)) // atomic
	return ret, nil
}
