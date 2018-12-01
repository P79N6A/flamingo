package grass

import (
	"bytes"
	"net"
	"time"

	"github.com/tinylib/msgp/msgp"
)

const (
	_SOCKET_PATH = "/opt/tmp/sock/grass.seqpacket.sock"
)

type GrassClient struct {
	conn          net.Conn
	internelQueue chan LogEntry
	errQueue      chan error
}

func (gc *GrassClient) Plant(hostname, category, body string) error {
	select {
	case gc.internelQueue <- LogEntry{Hostname: hostname, Category: category, Message: body}:
	case err := <-gc.errQueue:
		return err
	default:
	}
	return nil
}

func (gc *GrassClient) Close() error {
	return gc.conn.Close()
}

func (gc *GrassClient) recordErr(err error) {
	select {
	case gc.errQueue <- err:
	default:
	}
}

func (gc *GrassClient) process() {
	buf := bytes.NewBuffer(make([]byte, 65536))
	for {
		conn, err := net.DialUnix("unixpacket", nil, &net.UnixAddr{_SOCKET_PATH, "unixpacket"})
		if err != nil {
			gc.recordErr(err)
			time.Sleep(5 * time.Second)
			continue
		}
		gc.conn = conn
		for entry := range gc.internelQueue {
			buf.Reset()
			err := msgp.Encode(buf, entry)
			if err != nil {
				gc.recordErr(err)
				continue
			}
			conn.SetDeadline(time.Now().Add(time.Second))
			_, err = conn.Write(buf.Bytes())
			if err != nil {
				gc.recordErr(err)
				break
			}
		}
	}
}

func NewGrassClient() (*GrassClient, error) {
	gc := &GrassClient{
		internelQueue: make(chan LogEntry, 1024),
		errQueue:      make(chan error, 10),
	}
	go gc.process()
	return gc, nil
}
