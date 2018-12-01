package handler

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"code.bean.com/flamingo/service"
	"code.byted.org/gopkg/logs"
	"github.com/gin-gonic/gin"
)

// Handler 通用接口，提供注册函数
type Handler interface {
	Register(e *gin.Engine)
}

var (
	handlers      []Handler
	configService *service.ConfigService
)

const (
	// AKURL AccessToken url
	AKURL = "https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=wxe3811b0b04d53337&secret=c383e3f539137e3da5591320e20e67ad"
)

// JSONHandlerFunc 返回json结果的处理函数
type JSONHandlerFunc func(*gin.Context) (interface{}, error)

// AccessTokenResp access token返回结构体
type AccessTokenResp struct {
	AccessToken string `json:"access_token"`
	Expires     int    `json:"expires_in"`
}

// Init 初始化Handler层
func Init() {
	configService = service.NewConfigService()
	handlers = make([]Handler, 0)
	handlers = append(handlers, NewTemplateHandler(), NewWXAccessHandler(), NewCustomerHandler(), NewOperatorHandler())
	// go RefreshAccessToken()
}

// RegisterHandler 注册所有Handler的api接口
func RegisterHandler(e *gin.Engine) {
	for _, h := range handlers {
		h.Register(e)
	}
}

// RefreshAccessToken 微信access token
func RefreshAccessToken() {
	for {
		req, err := http.NewRequest("GET", AKURL, nil)
		if err != nil {
			logs.Fatalf("get access token req error:%+v\n", err)
		}
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			logs.Fatalf("get access token resp error:%+v\n", err)
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logs.Fatalf("read access token resp error:%+v\n", err)
		}
		logs.Info("refresh access token resp:%s", string(body))
		var token AccessTokenResp
		err = json.Unmarshal(body, &token)
		if err != nil || token.AccessToken == "" {
			continue
		}
		configService.UpdateAccessToken(token.AccessToken)
		logs.Info("update access token:%s", token.AccessToken)
		time.Sleep(7000 * time.Second)
	}

}

// JSONWrapper 将一个函数的返回结果用统一的json格式封装
func JSONWrapper(fn JSONHandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		logs.Debug("@@request is +%v", c.Request)
		data, err := fn(c)
		logs.Debug("@@data and err is +%v  %v", data, err)
		if err != nil {
			logs.Error("error: %v, path: %v, params: %v", err, c.Request.URL, c.Request.Form)
			se, ok := err.(*service.Error)
			if ok {
				c.JSON(http.StatusOK, se)
				return
			}
			e := service.ErrorWrap(service.StatusInternalServerError, err)
			c.JSON(http.StatusInternalServerError, e)
			return
		}

		c.JSON(http.StatusOK, map[string]interface{}{
			"code": 0,
			"data": data,
		})
	}
}
