package handler

import (
	"code.bean.com/flamingo/config"
	"code.bean.com/flamingo/service"
	"code.bean.com/flamingo/util"
	"code.byted.org/gopkg/logs"
	"github.com/gin-gonic/gin"
)

type OperatorHandler struct{}

func NewOperatorHandler() *OperatorHandler {
	return &OperatorHandler{}
}

func (handler *OperatorHandler) Register(e *gin.Engine) {
	group := e.Group("/operator")
	group.POST("/login", JSONWrapper(handler.Login))
	group.POST("/info", OperatorInfoMiddleware(), JSONWrapper(handler.OperatorInfo))
	group.POST("/add_customer", OperatorInfoMiddleware(), JSONWrapper(handler.AddNewCustomer))
	group.POST("/query_customer", OperatorInfoMiddleware(), JSONWrapper(handler.GetCustomerInfo))
	group.POST("/operate_customer", OperatorInfoMiddleware(), JSONWrapper(handler.OperateCustomer))
}

func (handler *OperatorHandler) Login(c *gin.Context) (interface{}, error) {
	phone := c.PostForm("cell")
	pwd := c.PostForm("pwd")
	if phone == "" || pwd == "" {
		return nil, service.NewError(401, "缺少必要参数")
	}
	operator, err := service.OperatorServiceInstance().OperatorLogin(phone, pwd)
	if err != nil {
		return nil, service.NewError(402, "密码错误")
	}
	enbytes, _ := util.AESEncrypt([]byte(operator.Cellphone))
	host, _ := config.ConfigJson.Get("host").String()
	c.SetCookie("operator_id", string(enbytes), 3600*24, "/", host, false, false)
	return "success", nil
}

func (handlers *OperatorHandler) OperatorInfo(c *gin.Context) (interface{}, error) {
	op, err := OperatorInfo(c)
	if err != nil {
		logs.Error("invalid user,err=%+v", err)
		return nil, err
	}
	return op.Name, err
}

func (handler *OperatorHandler) GetCustomerInfo(c *gin.Context) (interface{}, error) {
	phone := c.PostForm("cell")
	if phone == "" {
		return nil, service.NewError(401, "缺少必要参数")
	}
	return service.CustomerServiceInstance().GetCustomerDetailInfo(phone)
}

func (handler *OperatorHandler) AddNewCustomer(c *gin.Context) (interface{}, error) {
	phone := c.PostForm("cell")
	name := c.PostForm("name")
	return service.CustomerServiceInstance().CreateCustomer(phone, name)
}

func (handler *OperatorHandler) OperateCustomer(c *gin.Context) (interface{}, error) {
	op, err := OperatorInfo(c)
	if err != nil {
		logs.Error("invalid user,err=%+v", err)
		return nil, err
	}
	phone := c.PostForm("cell")
	oper := c.PostForm("operate_type")
	money := c.PostForm("amount")
	desc := c.PostForm("desc")
	if phone == "" || oper == "" || money == "" {
		return nil, service.NewError(401, "缺少必要参数")
	}
	return service.CustomerServiceInstance().AddCustomerAccount(phone, oper, money, desc, op)
}
