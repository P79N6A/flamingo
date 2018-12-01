// Package accesslog implements a middleware that emit an access log for each request.
//
// Includes fields like:
//   - status
//   - method
//   - client ip
//   - latency
package accesslog

import (
	"os"
	"time"

	"code.byted.org/gin/ginex/internal"
	internal_util "code.byted.org/gin/ginex/internal/util"
	"code.byted.org/gopkg/logs"
	"github.com/gin-gonic/gin"
)

func AccessLog(logger *logs.Logger) gin.HandlerFunc {
	psm := os.Getenv(internal.GINEX_PSM)
	cluster := internal_util.LocalCluster()
	return func(c *gin.Context) {
		if logger == nil {
			// 不能初始化log不是critical error, 并不影响正常运行
			c.Next()
			return
		}

		// some evil middlewares modify this values
		path := c.Request.URL.Path
		start := time.Now()
		c.Next()
		end := time.Now()
		latency := end.Sub(start).Nanoseconds() / 1000
		localIp := c.Value(internal.LOCALIPKEY)
		// status, latency, method, path, remote_ip, psm, log_id, local_cluster, host, user_agent
		logger.Trace("%s %s %s %s status=%d cost=%d method=%s full_path=%s client_ip=%s host=%s",
			localIp, psm, c.Value(internal.LOGIDKEY), cluster, c.Writer.Status(), latency,
			c.Request.Method, path, c.ClientIP(), c.Request.Host)
	}
}
