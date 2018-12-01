package kitc

import (
	"context"
	"io"
	"net"

	"code.byted.org/gopkg/thrift"
	"code.byted.org/kite/kitutil"
)

const (
	bufferedTransportLen = 4096
)

type stringWriter interface {
	WriteString(s string) (int, error)
}

// TODO(xiangchao): Conn() net.Conn method should be added ?
type Transport interface {
	io.ReadWriteCloser
	io.ByteReader
	io.ByteWriter
	stringWriter
	Open() error
	IsOpen() bool
	Flush() error
	RemoteAddr() string
	OpenWithContext(ctx context.Context) error
}

func NewBufferedTransport(kc *KitcClient) Transport {
	return &BufferedTransport{client: kc}
}

// BufferedTransport implement thrift.TRichTransport
type BufferedTransport struct {
	trans  *thrift.TBufferedTransport
	conn   net.Conn
	client *KitcClient
}

// RemoteAddr
func (bt *BufferedTransport) RemoteAddr() string {
	if bt.conn != nil {
		return bt.conn.RemoteAddr().String()
	}
	return ""
}

func (bt *BufferedTransport) Read(p []byte) (int, error) {
	return bt.trans.Read(p)
}

func (bt *BufferedTransport) Write(p []byte) (int, error) {
	return bt.trans.Write(p)
}

func (bt *BufferedTransport) ReadByte() (byte, error) {
	return bt.trans.ReadByte()
}

func (bt *BufferedTransport) WriteByte(c byte) error {
	return bt.trans.WriteByte(c)
}

func (bt *BufferedTransport) WriteString(s string) (int, error) {
	return bt.trans.WriteString(s)
}

func (bt *BufferedTransport) Flush() error {
	return bt.trans.Flush()
}

func (bt *BufferedTransport) OpenWithContext(ctx context.Context) error {
	conn, ok := kitutil.GetCtxTargetConn(ctx)
	if ok == false || conn == nil {
		return Err("No target connection in the context")
	}

	socket := GetSocketWithContext(conn, ctx)
	bt.trans = thrift.NewTBufferedTransport(socket, bufferedTransportLen)
	bt.conn = conn
	return nil
}

func (bt *BufferedTransport) Open() error {
	return bt.trans.Open()
}

func (bt *BufferedTransport) IsOpen() bool {
	return bt.trans.IsOpen()
}

func (bt *BufferedTransport) Close() error {
	return bt.trans.Close()
}

// NewFramedTransport return a FramedTransport
func NewFramedTransport(kc *KitcClient) Transport {
	return &FramedTransport{client: kc}
}

// FramedTransport implement thrift.TRichTransport
type FramedTransport struct {
	trans  *thrift.TFramedTransport
	conn   net.Conn
	client *KitcClient
}

func (ft *FramedTransport) RemoteAddr() string {
	if ft.conn != nil {
		return ft.conn.RemoteAddr().String()
	}
	return ""
}

func (ft *FramedTransport) Read(p []byte) (int, error) {
	return ft.trans.Read(p)
}

func (ft *FramedTransport) Write(p []byte) (int, error) {
	return ft.trans.Write(p)
}

func (ft *FramedTransport) ReadByte() (byte, error) {
	return ft.trans.ReadByte()
}

func (ft *FramedTransport) WriteByte(c byte) error {
	return ft.trans.WriteByte(c)
}

func (ft *FramedTransport) WriteString(s string) (int, error) {
	return ft.trans.WriteString(s)
}

func (ft *FramedTransport) Flush() error {
	return ft.trans.Flush()
}

func (ft *FramedTransport) Open() error {
	return ft.trans.Open()
}

// OpenWithContext connect a backend server acording the content of ctx
func (ft *FramedTransport) OpenWithContext(ctx context.Context) error {
	conn, ok := kitutil.GetCtxTargetConn(ctx)
	if ok == false || conn == nil {
		return Err("No target connection in the context")
	}

	socket := GetSocketWithContext(conn, ctx)
	bt := thrift.NewTBufferedTransport(socket, bufferedTransportLen)
	ft.trans = thrift.NewTFramedTransport(bt)
	ft.conn = conn
	return nil
}

func (ft *FramedTransport) IsOpen() bool {
	return ft.trans.IsOpen()
}

func (ft *FramedTransport) Close() error {
	return ft.trans.Close()
}
