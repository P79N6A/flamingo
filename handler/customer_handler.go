package handler

import (
	"code.bean.com/flamingo/config"
	"code.bean.com/flamingo/service"
	"code.bean.com/flamingo/util"
	"code.byted.org/gopkg/logs"
	"github.com/gin-gonic/gin"
)

type CustomersHandler struct{}

func NewCustomerHandler() *CustomersHandler {
	return &CustomersHandler{}
}

func (handler *CustomersHandler) Register(e *gin.Engine) {
	group := e.Group("/cu")
	group.POST("/check_code", JSONWrapper(handler.SendCheckCode))
	group.POST("login", JSONWrapper(handler.Login))
	group.POST("/cu_detail", CustomersInfoMiddleware(), JSONWrapper(handler.GetCustomerInfo))
}

func (handler *CustomersHandler) SendCheckCode(c *gin.Context) (interface{}, error) {
	phone := c.PostForm("cell")
	if phone == "" {
		return nil, service.NewError(401, "缺少必要参数")
	}
	err := service.CustomerServiceInstance().SendCheckCode(phone)
	if err != nil {
		return nil, err
	}
	return "success", nil
}

func (handler *CustomersHandler) Login(c *gin.Context) (interface{}, error) {
	// phone := c.PostForm("cell")
	// pwd := c.PostForm("pwd")
	phone := c.PostForm("cell")
	code := c.PostForm("code")
	verify, err := service.CustomerServiceInstance().VerifyCheckCode(code, phone)
	if err != nil {
		return nil, err
	}
	if !verify {
		return nil, service.ErrPasswordCheckCodeNotMatch
	}
	enbytes, _ := util.AESEncrypt([]byte(phone))
	host, _ := config.ConfigJson.Get("host").String()
	c.SetCookie("customer_id", string(enbytes), 3600*24, "/", host, false, false)
	return "success", nil
}

func (handler *CustomersHandler) GetCustomerInfo(c *gin.Context) (interface{}, error) {
	customer, err := CustomerInfo(c)
	if err != nil {
		logs.Error("invalid user,err=%+v", err)
		return nil, service.ErrorServiceInternalError
	}
	return service.CustomerServiceInstance().GetCustomerDetailInfo(customer.Cellphone)
}
