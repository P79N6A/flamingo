package nsq

import (
	"container/list"
	"net"
	"net/http"
	"time"
)

var (
	logger         Logger
	ConnectTimeout = 3 * time.Second
	ReadTimeout    = 5 * time.Second
	client         *http.Client
	LimitConns     = 10000
	Limit          chan struct{}
	PerHostConns   = 10
	Interval       = 5
)

func init() {
	InitClient()
}

func InitClient() {
	client = &http.Client{
		Transport: &http.Transport{
			Dial: func(network, addr string) (net.Conn, error) {
				return NewHttpConn(network, addr)
			},
			MaxIdleConnsPerHost: PerHostConns,
		},
		Timeout: ReadTimeout,
	}
	Limit = make(chan struct{}, LimitConns)
	logger = new(defaultLogger)
}

func SetLogger(lg Logger) {
	logger = lg
}

type LimitConnErr struct{}

func (le *LimitConnErr) Error() string {
	return "LimitConnErr"
}

type HttpConn struct {
	net.Conn
}

func NewHttpConn(network, addr string) (*HttpConn, error) {
	select {
	case Limit <- struct{}{}:
		conn, err := net.DialTimeout(network, addr, ConnectTimeout)
		if err != nil {
			<-Limit
			return nil, err
		}
		return &HttpConn{conn}, err
	default:
	}
	return nil, &LimitConnErr{}
}

func (hc *HttpConn) Close() error {
	err := hc.Conn.Close()
	<-Limit
	return err
}

type NsqPool struct {
	addrList []string
	works    *list.List
	ch       chan *MsgEntry
	multi    int
	stop     chan struct{}
	invoker  Invoker
}

func NewNsqPoolWithInvoker(addrlist []string, ch chan *MsgEntry, multi int, invoker Invoker) *NsqPool {
	return &NsqPool{
		addrList: addrlist,
		works:    list.New(),
		ch:       ch,
		multi:    multi,
		stop:     make(chan struct{}),
		// for compatible, you must use SetInvoker to set invoker
		invoker: invoker,
	}
}

func NewNsqPool(addrlist []string, ch chan *MsgEntry, multi int) *NsqPool {
	return &NsqPool{
		addrList: addrlist,
		works:    list.New(),
		ch:       ch,
		multi:    multi,
		stop:     make(chan struct{}),
		// for compatible, you must use SetInvoker to set invoker
		invoker: new(FooInvoker),
	}
}

func (np *NsqPool) SetInvoker(invoker Invoker) {
	np.invoker = invoker
}

func (np *NsqPool) Handle() {
	checker := time.NewTicker(time.Duration(Interval) * time.Second)
	for {
		select {
		case <-checker.C:
			np.CheckWorkers()
		case <-np.stop:
			for e := np.works.Front(); e != nil; e = e.Next() {
				e.Value.(*Worker).Stop()
			}
			checker.Stop()
			return
		}
	}
}

func (np *NsqPool) Stop() {
	np.stop <- struct{}{}
}

func (np *NsqPool) CheckWorkers() {
	for e := np.works.Front(); e != nil; e = e.Next() {
		e.Value.(*Worker).Run()
	}
}

func (np *NsqPool) Start() {
	for _, addr := range np.addrList {
		for i := 0; i < np.multi; i++ {
			w := NewWorker(addr, np.ch)
			w.SetInvoker(np.invoker)
			np.AddWork(w)
		}
	}
	np.CheckWorkers()
}

func (np *NsqPool) AddWork(w *Worker) {
	np.works.PushFront(w)
}

type MsgEntry struct {
	Topic string
	Defer int64 // in milliseconds, refer https://code.byted.org/snippets/39
	Msg   []byte
}
