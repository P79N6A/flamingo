package throttle

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

const (
	DEFAULT_QPS_LIMIT       = 10000
	DEFAULT_BURST_QPS_LIMIT = 20000
)

var (
	limiter = rate.NewLimiter(DEFAULT_QPS_LIMIT, DEFAULT_BURST_QPS_LIMIT)
)

func Throttle() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.AbortWithStatus(http.StatusTooManyRequests)
		}
		c.Next()
	}
}
