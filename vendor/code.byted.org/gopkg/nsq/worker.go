package nsq

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

type NsqPutter struct {
	addr string
}

func NewNsqPutter(addr string) *NsqPutter {
	return &NsqPutter{addr: addr}
}

func (n *NsqPutter) Send(topic string, message []byte) error {
	return n.send(topic, message, 0)
}

func (n *NsqPutter) send(topic string, message []byte, deferTime int64) error {
	url := fmt.Sprintf("%s/put?topic=%s&defer=%d", n.addr, topic, deferTime)
	resp, err := client.Post(url, "plain/text", bytes.NewReader(message))
	if err == nil {
		defer resp.Body.Close()
		ioutil.ReadAll(resp.Body) // 清空body的数据, 用于connc的复用
		if resp.StatusCode != 200 {
			return fmt.Errorf("send to nsq %s error resp code %d", n.addr, resp.StatusCode)
		}
	}
	return err
}

type Worker struct {
	addr    string // 用于标记当前worker
	isRun   int32
	ch      chan *MsgEntry
	stop    chan struct{}
	sender  *NsqPutter
	invoker Invoker
}

func NewWorker(addr string, ch chan *MsgEntry) *Worker {
	sender := NewNsqPutter(addr)
	return &Worker{
		addr:   addr,
		isRun:  0,
		ch:     ch,
		stop:   make(chan struct{}),
		sender: sender,
	}
}

func (w *Worker) SetInvoker(invoker Invoker) {
	w.invoker = invoker
}

func (w *Worker) loop() {
	defer func() {
		if err := recover(); err != nil {
			const size = 64 << 20
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			logger.Error("worker %s panic :%v\n%s\n", w.addr, err, buf)
		}
		atomic.StoreInt32(&w.isRun, 0)
	}()
	for {
		select {
		case <-w.stop:
			return
		case entry := <-w.ch:
			err := w.sender.send(entry.Topic, entry.Msg, entry.Defer)
			if err != nil {
				if w.invoker != nil {
					w.invoker.Err(entry, err)
				}
				if strings.Contains(err.Error(), "LimitConnErr") {
					time.Sleep(5 * time.Millisecond)
					logger.Warn("Reach limitConn %d", LimitConns)
				} else {
					logger.Error("Send topic %s error %s", entry.Topic, err)
					return
				}
			} else {
				if w.invoker != nil {
					w.invoker.Succ()
				}
			}
		}
	}
}

func (w *Worker) Run() error {
	if atomic.CompareAndSwapInt32(&w.isRun, 0, 1) {
		go w.loop()
	}
	return nil
}

func (w *Worker) IsRun() bool {
	return atomic.LoadInt32(&w.isRun) == 1
}

func (w *Worker) Stop() error {
	select {
	case w.stop <- struct{}{}:
	case <-time.After(time.Second):
		return errors.New("stop worker time out.")
	}
	return nil
}
