package ginex

import (
	"code.byted.org/gin/ginex/internal"
	"code.byted.org/gopkg/context"
	"github.com/gin-gonic/gin"
)

// Deprecated: use RpcContext
func MethodContext(ginCtx *gin.Context) context.Context {
	return RpcContext(ginCtx)
}

// RpcContext returns context information for loanrpc call
// It's a good choice to call this method at the beginning of handler function to
// avoid concurrent read and write gin.Context
func RpcContext(ginCtx *gin.Context) context.Context {
	ctx := context.WithValue(context.Background(), internal.SNAMEKEY, PSM())
	ctx = context.WithValue(ctx, internal.LOCALIPKEY, LocalIP())
	ctx = context.WithValue(ctx, internal.CLUSTERKEY, LocalCluster())
	if logID, exist := ginCtx.Get(internal.LOGIDKEY); exist {
		ctx = context.WithValue(ctx, internal.LOGIDKEY, logID.(string))
	}
	if method, exist := ginCtx.Get(internal.METHODKEY); exist {
		ctx = context.WithValue(ctx, internal.METHODKEY, method.(string))
	}
	return ctx
}
