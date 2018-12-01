package databus_client

import (
	"code.byted.org/golf/metrics"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// 1. 尽量复用DatabusCollector对象
// 2. 测试情况下BufferedDatabusCollector吞吐会比DatabusCollector吞吐低
// 3. 如果DatabusCollector对象太多会导致domain socket连接特别多

const (
	DEFAULT_SOCKET_PATH  = "/opt/tmp/sock/databus_collector.seqpacket.sock"
	PACKET_SIZE_LIMIT    = 212992 // 208KB
	DEFAULT_TIMEOUT      = 100 * time.Millisecond
	ERROR_TOLERATE       = 50
	DEFAULT_MAX_CONN_NUM = 5
	READ_BUFFER_SIZE     = 1 << 10
	METRICS_PREFIX       = "inf.databus"
	METRICS_SUCC         = "collect.success"
	METRICS_FAIL         = "collect.fail"
	RESP_SUCC_CODE       = 0
)

var (
	ErrAlreadyClosed       = errors.New("client already closed")
	ErrResponseCodeNotSucc = errors.New("databus response code said not succ")
	cm                     = newClientMetrics()
)

type clientMetrics struct {
	definedChannels map[string]*metricsCounter
	rwmu            sync.RWMutex
	tagkv           map[string]string // to avoid gc
}

type metricsCounter struct {
	succCounter int64
	failCounter int64
}

func newClientMetrics() *clientMetrics {
	metrics.InitWithDefaultServer(METRICS_PREFIX, true)
	metrics.DefineCounter(METRICS_SUCC, "")
	metrics.DefineCounter(METRICS_FAIL, "")
	definedChannels := make(map[string]*metricsCounter)
	tagkv := make(map[string]string)
	cm := &clientMetrics{definedChannels: definedChannels, tagkv: tagkv}
	go cm.metricsRoutine()
	return cm
}

func (cm *clientMetrics) AddCounter(channel string, success bool) {
	cm.rwmu.RLock()
	mc, ok := cm.definedChannels[channel]
	cm.rwmu.RUnlock()
	if !ok {
		cm.rwmu.Lock()
		mc, ok = cm.definedChannels[channel]
		if !ok {
			mc = &metricsCounter{}
			cm.definedChannels[channel] = mc
			tagvs := make([]string, 1, 1)
			tagvs[0] = channel
			metrics.DefineTagkv("channel", tagvs)
		}
		cm.rwmu.Unlock()
	}
	if success {
		atomic.AddInt64(&mc.succCounter, 1)
	} else {
		atomic.AddInt64(&mc.failCounter, 1)
	}
}

func (cm *clientMetrics) metricsRoutine() {
	// because cm.definedChannels is not delete, so we let the cow object outside to avoid gc
	definedChannels := make(map[string]*metricsCounter)
	for {
		select {
		case <-time.After(time.Second * 1):
			cm.rwmu.RLock()
			for channel, mc := range cm.definedChannels {
				definedChannels[channel] = mc
			}
			cm.rwmu.RUnlock()
			for channel, mc := range definedChannels {
				succCounter := atomic.SwapInt64(&mc.succCounter, 0)
				failCounter := atomic.SwapInt64(&mc.failCounter, 0)
				cm.emitSucc(channel, succCounter)
				cm.emitFail(channel, failCounter)
			}
		}
	}

}

func (cm *clientMetrics) emitSucc(channel string, succCounter int64) {
	cm.emitCounter(channel, METRICS_SUCC, succCounter)
}

func (cm *clientMetrics) emitFail(channel string, failCounter int64) {
	cm.emitCounter(channel, METRICS_FAIL, failCounter)
}

func (cm *clientMetrics) emitCounter(channel string, name string, value int64) {
	cm.tagkv["channel"] = channel
	valueStr := strconv.FormatInt(value, 10)
	metrics.EmitCounter(name, valueStr, "", cm.tagkv)
}

type conn struct {
	nc       *net.UnixConn
	config   *DatabusCollector
	errTimes int
}

func (cn *conn) Write(b []byte) error {
	// 发送失败次数达到阈值重建链接

	if cn.nc == nil || cn.errTimes > ERROR_TOLERATE {
		cn.errTimes = 0
		con, err := net.DialUnix("unixpacket", nil, &net.UnixAddr{cn.config.socketPath, "unixpacket"})
		if err != nil {
			return err
		}
		if cn.nc != nil {
			cn.nc.Close()
		}
		cn.nc = con
	}
	cn.nc.SetDeadline(time.Now().Add(cn.config.writeTimeout))
	_, err := cn.nc.Write(b)
	if err != nil {
		cn.errTimes += 1
		return err
	}
	return nil
}

func (cn *conn) Read(buf []byte) (n int, err error) {
	n, err = cn.nc.Read(buf)
	if err != nil {
		cn.errTimes += 1
	}
	return
}

func (cn *conn) release() {
	if cn.nc != nil {
		cn.nc.Close()
	}
}

type DatabusCollector struct {
	socketPath   string
	pool         chan *conn
	writeTimeout time.Duration
}

func NewDefaultCollector() *DatabusCollector {
	return NewCollector(DEFAULT_SOCKET_PATH, DEFAULT_TIMEOUT, DEFAULT_MAX_CONN_NUM)
}

func NewCollectorWithTimeout(timeout time.Duration) *DatabusCollector {
	return NewCollector(DEFAULT_SOCKET_PATH, timeout, DEFAULT_MAX_CONN_NUM)
}

func NewCollector(socketPath string, timeout time.Duration, maxConn int) *DatabusCollector {
	collector := &DatabusCollector{
		socketPath:   socketPath,
		writeTimeout: timeout,
		pool:         make(chan *conn, maxConn),
	}
	return collector
}

// 从pool中取出一个连接
func (this *DatabusCollector) borrow() (*conn, error) {
	var cl *conn
	var ok bool
	select {
	case cl, ok = <-this.pool:
		if !ok {
			return nil, ErrAlreadyClosed
		}
	default:
		cl = this.newConn()
		fmt.Println("pool empty create a databus connection")
	}
	return cl, nil
}

// 归还连接
func (this *DatabusCollector) putBack(cl *conn) {
	select {
	case this.pool <- cl:
	default:
		cl.release()
		fmt.Println("pool full destory a databus connection")
	}
}

func (this *DatabusCollector) newConn() *conn {
	con := &conn{
		nc:       nil,
		config:   this,
		errTimes: 0,
	}
	return con
}

func (this *DatabusCollector) CollectArray(channel string, messages []*ApplicationMessage) error {
	var payload RequestPayload
	payload.Channel = &channel
	payload.Messages = messages
	buf, err := proto.Marshal(&payload)
	defer func() {
		this.EmitMetric(channel, err)
	}()
	if err != nil {
		return err
	}
	if len(buf) > PACKET_SIZE_LIMIT {
		err = fmt.Errorf("CollectArray Err: Packet too large.")
		return err
	}
	con, err := this.borrow()
	if err != nil {
		return err
	}
	defer this.putBack(con)
	err = con.Write(buf)
	if err != nil {
		return err
	}
	return nil
}

func (this *DatabusCollector) CollectArrayWithResp(channel string, messages []*ApplicationMessage) (*ResponsePayload, error) {
	var payload RequestPayload
	payload.Channel = &channel
	payload.Messages = messages
	payload.NeedResp = proto.Int32(1)
	buf, err := proto.Marshal(&payload)
	defer func() {
		this.EmitMetric(channel, err)
	}()
	if err != nil {
		return nil, err
	}
	if len(buf) > PACKET_SIZE_LIMIT {
		err = fmt.Errorf("CollectArray Err: Packet too large.")
		return nil, err
	}
	con, err := this.borrow()
	if err != nil {
		return nil, err
	}
	defer this.putBack(con)
	err = con.Write(buf)
	if err != nil {
		return nil, err
	}
	resp, err := this.recvResp(con)
	if err != nil {
		return nil, err
	}
	if resp.GetCode() != RESP_SUCC_CODE {
		err = ErrResponseCodeNotSucc
	}
	return resp, err
}

func (this *DatabusCollector) Collect(channel string, value []byte, key []byte, codec int32) error {
	var message ApplicationMessage
	message.Key = key
	message.Value = value
	message.Codec = &codec
	var payload RequestPayload
	payload.Channel = &channel
	payload.Messages = []*ApplicationMessage{&message}
	buf, err := proto.Marshal(&payload)
	defer func() {
		this.EmitMetric(channel, err)
	}()
	if err != nil {
		return err
	}
	con, err := this.borrow()
	if err != nil {
		return err
	}
	err = con.Write(buf)
	this.putBack(con)
	if err != nil {
		return err
	}
	return nil
}

func (this *DatabusCollector) CollectWithResp(channel string, value []byte, key []byte, codec int32) (*ResponsePayload, error) {
	var message ApplicationMessage
	message.Key = key
	message.Value = value
	message.Codec = &codec
	var payload RequestPayload
	payload.Channel = &channel
	payload.NeedResp = proto.Int32(1)
	payload.Messages = []*ApplicationMessage{&message}
	buf, err := proto.Marshal(&payload)
	defer func() {
		this.EmitMetric(channel, err)
	}()
	if err != nil {
		return nil, err
	}
	con, err := this.borrow()
	if err != nil {
		return nil, err
	}
	defer this.putBack(con)
	err = con.Write(buf)
	if err != nil {
		return nil, err
	}
	resp, err := this.recvResp(con)
	if err != nil {
		return nil, err
	}
	if resp.GetCode() != RESP_SUCC_CODE {
		err = ErrResponseCodeNotSucc
	}
	return resp, err
}

// 内部实现，不要调用
func (this *DatabusCollector) SendProto(b []byte) error {
	con, err := this.borrow()
	if err != nil {
		return err
	}
	err = con.Write(b)
	this.putBack(con)
	if err != nil {
		return err
	}
	return nil
}

// 内部实现，不要调用
func (this *DatabusCollector) SendProtoRecvResp(b []byte) (*ResponsePayload, error) {
	con, err := this.borrow()
	if err != nil {
		return nil, err
	}
	defer this.putBack(con)
	err = con.Write(b)
	if err != nil {
		return nil, err
	}
	resp, err := this.recvResp(con)
	return resp, err
}

func (this *DatabusCollector) recvResp(con *conn) (*ResponsePayload, error) {
	buf := make([]byte, READ_BUFFER_SIZE)
	n, err := con.Read(buf)
	if err != nil {
		return nil, err
	}
	resp, err := this.extractResp(buf, n)
	return resp, err
}

func (this *DatabusCollector) extractResp(b []byte, n int) (*ResponsePayload, error) {
	resp := &ResponsePayload{}
	err := proto.Unmarshal(b[0:n], resp)
	if err != nil {
		return nil, err
	}
	return resp, err
}

func (this *DatabusCollector) Close() error {
	//连接等待gc关闭, close queue 会触发panic
	return nil
}

func (this *DatabusCollector) EmitMetric(channel string, err error) {
	cm.AddCounter(channel, err == nil)
}
