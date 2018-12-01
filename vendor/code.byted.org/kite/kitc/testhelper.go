package kitc

import (
	"encoding/json"
	"net"
	"strings"
	"time"

	"code.byted.org/kite/endpoint"
	"code.byted.org/kite/kitutil"
	"code.byted.org/kite/kitware"
)

type mockETCD struct {
	switchVal      string
	circuitbreaker string
	configVal      string
}

func (me *mockETCD) Get(key string) (interface{}, error) {
	if strings.Contains(key, "switch") {
		return me.switchVal, nil
	}
	if strings.Contains(key, "circuitbreaker") {
		return me.circuitbreaker, nil
	}
	return "", nil
}

type mockClient struct {
	Caller *mockCaller
}

func (mc *mockClient) New(kc *KitcClient) Caller { return mc.Caller }

type mockRequest struct{}

func (mr *mockRequest) SetBase(kb endpoint.KiteBase) error { return nil }
func (mr *mockRequest) RealRequest() interface{}           { return nil }

type mockCaller struct {
	EndPoint endpoint.EndPoint
	Request  endpoint.KitcCallRequest
}

func (mc *mockCaller) Call(name string, request interface{}) (endpoint.EndPoint, endpoint.KitcCallRequest) {
	return mc.EndPoint, mc.Request
}

type mockACLer struct {
	ACL string
}

func (ma *mockACLer) GetByKey(key string) (string, error) { return ma.ACL, nil }

type mockDegradater struct {
	percent int
	random  int
}

func (md *mockDegradater) GetDegradationPercent(key string) (int, error) { return md.percent, nil }
func (md *mockDegradater) RandomPercent() int                            { return md.random }

type mockIDCSelector struct {
	idc string
}

func (mis *mockIDCSelector) SelectIDC(policies []kitutil.TPolicy) (string, error) { return mis.idc, nil }

type mockDiscoverer struct {
	name string
	ins  []kitutil.Instance
}

func (md *mockDiscoverer) Name() string { return md.name }

func (md *mockDiscoverer) Lookup(idc, cluster, env string) ([]kitutil.Instance, error) {
	return md.ins, nil
}

type mockInstance struct {
	host string
	port string
	tags map[string]string
}

func (mi *mockInstance) Host() string            { return mi.host }
func (mi *mockInstance) Port() string            { return mi.port }
func (mi *mockInstance) Tags() map[string]string { return mi.tags }

type mockConn struct{}

func (mc *mockConn) Read(b []byte) (n int, err error)   { return 0, nil }
func (mc *mockConn) Write(b []byte) (n int, err error)  { return len(b), nil }
func (mc *mockConn) Close() error                       { return nil }
func (mc *mockConn) LocalAddr() net.Addr                { return nil }
func (mc *mockConn) RemoteAddr() net.Addr               { return nil }
func (mc *mockConn) SetDeadline(t time.Time) error      { return nil }
func (mc *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (mc *mockConn) SetWriteDeadline(t time.Time) error { return nil }

type mockPooler struct {
	conn *mockConn
}

func (mp *mockPooler) Get(ins kitutil.Instance, timeout time.Duration) (net.Conn, error) {
	return mp.conn, nil
}

type mockDyconfiger struct {
	Policy kitware.RPCPolicy
}

func newMockDyconfiger() *mockDyconfiger {
	policy := kitware.RPCPolicy{
		RetryTimes:          0,
		ConnectTimeout:      20,
		ConnectRetryMaxTime: 3,
		ReadTimeout:         500,
		WriteTimeout:        500,
		TrafficPolicy:       []kitutil.TPolicy{kitutil.TPolicy{}},
	}

	return &mockDyconfiger{
		Policy: policy,
	}
}

func (md *mockDyconfiger) GetByKey(key string) ([]byte, error) {
	return json.Marshal(md.Policy)
}
