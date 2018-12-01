package kitc

import (
	"context"
	"net"
	"time"

	"code.byted.org/gopkg/logs"
	"code.byted.org/gopkg/thrift"
	"code.byted.org/kite/kitutil"
)

const (
	defaultReadWriteTimeout = 500 * time.Millisecond
)

// func GetConnWithContext(client *KitcClient, ctx context.Context) (net.Conn, error) {
// 	return client.Pool.Get(ctx)
// }

func GetSocketWithContext(conn net.Conn, ctx context.Context) *thrift.TSocket {
	timeout := defaultReadWriteTimeout
	deadline, ok := ctx.Deadline()
	if dur := deadline.Sub(time.Now()); ok && dur > 500*time.Millisecond {
		timeout = dur
	}

	targetService, _ := kitutil.GetCtxTargetServiceName(ctx)
	targetMethod, _ := kitutil.GetCtxTargetMethod(ctx)
	logs.Debugf("%s.%s readWriteTimeout is %s", targetService, targetMethod, timeout)
	return thrift.NewTSocketFromConnTimeout(conn, timeout)
}
