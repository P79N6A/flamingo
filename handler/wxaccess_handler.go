package handler

import (
	"crypto/sha1"
	"fmt"
	"io"
	"sort"
	"strings"

	"code.byted.org/gopkg/logs"
	"github.com/gin-gonic/gin"
)

const (
	Token = "hello2018"
)

// WXAccessHandler 微信通用认证api
type WXAccessHandler struct{}

// NewWXAccessHandler 实例化
func NewWXAccessHandler() *WXAccessHandler {
	return &WXAccessHandler{}
}

// Register 注册api
func (handler *WXAccessHandler) Register(e *gin.Engine) {
	group := e.Group("/wx")
	group.GET("/wx_conn", handler.FirstConn)
}

// FirstConn 首次连接
func (handler *WXAccessHandler) FirstConn(c *gin.Context) {
	// 注意下面将gin.H参数传入index.tmpl中!也就是使用的是index.tmpl模板
	sign, ts, nonce := c.Query("signature"), c.Query("timestamp"), c.Query("nonce")
	mySign := makeSignature(ts, nonce)
	if mySign == sign {
		echostr := c.Query("echostr")
		logs.Info("ECHO_STR:%s", echostr)
		c.Writer.Write([]byte(echostr))
	}
}

func makeSignature(timestamp, nonce string) string {
	s1 := []string{Token, timestamp, nonce}
	sort.Strings(s1)
	s := sha1.New()
	io.WriteString(s, strings.Join(s1, ""))
	return fmt.Sprintf("%x", s.Sum(nil))
}
