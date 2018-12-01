/*
 *  KITE RPC FRAMEWORK
 */

package kite

import (
	"fmt"
	"net"
	"runtime"
	"time"

	"code.byted.org/gopkg/logs"
	"code.byted.org/gopkg/thrift"
	"code.byted.org/kite/kitware"
)

type RpcServer struct {
	l net.Listener

	processorFactory thrift.TProcessorFactory
	transportFactory thrift.TTransportFactory
	protocolFactory  thrift.TProtocolFactory

	onConnect    func() error
	onProcess    func() error
	onDisconnect func()
	onPanic      func()
}

func NewRpcServer() *RpcServer {
	// Using buffered transport and binary protocol as default,
	// buffer size is 4096
	transport := thrift.NewTBufferedTransportFactory(DefaultTransportBufferedSize)
	protocol := thrift.NewTBinaryProtocolFactoryDefault()
	panicHooker := NewPanicHooker(panicFmt, ServiceName)
	limiter := NewEtcdLimiter(ServiceName)
	limitHooker := NewLimitHooker(int(limitQPS), int(limitMaxConns), limiter, ServiceName)
	return &RpcServer{
		transportFactory: transport,
		protocolFactory:  protocol,

		onConnect:    limitHooker.OnConnect,
		onProcess:    limitHooker.OnProcess,
		onDisconnect: limitHooker.OnDisconnect,
		onPanic:      panicHooker.OnPanic,
	}
}

func (p *RpcServer) Listen() error {
	l, err := net.Listen("tcp", ServiceAddr+ServicePort)
	if err != nil {
		return err
	}
	p.l = l
	return nil
}

func (p *RpcServer) AcceptLoop() error {
	for {
		// If l.Close() is called will return closed error
		conn, err := p.l.Accept()
		if err != nil {
			// wait most for most 5s
			deadline := time.After(5 * time.Second)
			for {
				select {
				case <-deadline:
					return err
				default:
					if CurrentConns() == 0 {
						return nil
					}
					time.Sleep(time.Millisecond)
				}
			}
		}
		logs.Debugf("KITE: remote address: %s", conn.RemoteAddr().String())
		if err := p.onConnect(); err != nil {
			logs.Error("KITE: %s", err)
			conn.Close()
			continue
		}
		if err := p.onProcess(); err != nil {
			logs.Error("KITE: %s, close socket forcely", err)
			conn.Close()
			p.onDisconnect()
			continue
		}
		go func(conn net.Conn) {
			client := thrift.NewTSocketFromConnTimeout(conn, ReadWriteTimeout)
			ip := conn.RemoteAddr().String()
			if err := p.processRequests(client); err != nil {
				logs.Warnf("KITE: processing request error=%s, remoteIP=%s", err, ip)
			}
		}(conn)
	}
}

func (p *RpcServer) Serve() error {
	if Processor == nil {
		logs.Fatal("KITE: Processor is nil")
		panic("KITE: Processor is nil")
	}
	logs.Info("KITE: server listen on %s", ServiceAddr+ServicePort)
	if err := p.Listen(); err != nil {
		return err
	}
	p.processorFactory = thrift.NewTProcessorFactory(Processor)
	if EnableMetrics {
		GoStatMetrics()
	}
	startDebugServer()
	if err := Register(); err != nil {
		logs.Fatal("KITE: Register service error: %s", err)
	}
	return p.AcceptLoop()
}

func (p *RpcServer) Stop() error {
	return p.l.Close()
}

func (p *RpcServer) processRequests(client thrift.TTransport) error {
	processor := p.processorFactory.GetProcessor(client)
	transport := p.transportFactory.GetTransport(client)
	protocol := p.protocolFactory.GetProtocol(transport)
	defer func() {
		if e := recover(); e != nil {
			if err, ok := e.(string); !ok || err != kitware.RecoverMW {
				const size = 64 << 10
				buf := make([]byte, size)
				buf = buf[:runtime.Stack(buf, false)]
				logs.Fatal("KITE: panic in processor: %s: %s", e, buf)
			}
			p.onPanic()
		}
		p.onDisconnect()
	}()
	defer transport.Close()
	// This loop for processing request on a connection.
	var count = 0
	for {
		count++
		metricsClient.EmitCounter("kite.process.throughput", 1, "", map[string]string{"name": ServiceName})
		ok, err := processor.Process(protocol, protocol)
		if err, ok := err.(thrift.TTransportException); ok {
			if err.TypeId() == thrift.END_OF_FILE ||
				// TODO(xiangchao.01): this timeout maybe not precision,
				// fix should in thrift package later.
				err.TypeId() == thrift.TIMED_OUT {
				return nil
			}
			if err.TypeId() == thrift.UNKNOWN_METHOD {
				name := fmt.Sprintf("toutiao.service.thrift.%s.process.error", ServiceName)
				metricsClient.EmitCounter(name, 1, "", map[string]string{
					"name": "UNKNOWN_METHOD",
				})
			}
		}

		if err != nil {
			return err
		}
		if !ok {
			break
		}
		// 当请求是短连接的时候，会在第二次循环的时候，读取到thrift.END_OF_FILE的错误
		// 为了兼容这种情况，当count等于1的时候，不应该执行onProcess调用
		if count == 1 {
			continue
		}
		if err := p.onProcess(); err != nil {
			logs.Error("KITE: %s, close socket forcely", err)
			return err
		}
	}
	return nil
}
