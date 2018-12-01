package handler

import (
	"net/http"

	"code.bean.com/flamingo/model"

	"code.bean.com/flamingo/util"

	"code.bean.com/flamingo/service"
	"github.com/gin-gonic/gin"
)

//CustomersInfoMiddleware 用户信息解析
func CustomersInfoMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		cellPhone, err := c.Cookie("customer_id")
		if err != nil || cellPhone == "" {
			getErrorResponse(c, http.StatusUnauthorized, service.ErrUserNotLogin)
			return
		}
		decriptBytes, err := util.AESDecrypt([]byte(cellPhone))
		if err != nil {
			getErrorResponse(c, http.StatusUnauthorized, service.ErrUserNotLogin)
			return
		}
		customer, err := model.CustomerDaoInstance().GetCustomerByCellphone(string(decriptBytes))
		if err != nil {
			getErrorResponse(c, http.StatusUnauthorized, service.ErrUserNotLogin)
			return
		}
		c.Set("customer", customer)
		c.Next()
		return
	}
}

//OperatorInfoMiddleware 操作员信息解析
func OperatorInfoMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		operatorID, err := c.Cookie("operator_id")
		if err != nil || operatorID == "" {
			getErrorResponse(c, http.StatusUnauthorized, service.ErrUserNotLogin)
			return
		}
		decriptBytes, err := util.AESDecrypt([]byte(operatorID))
		if err != nil {
			getErrorResponse(c, http.StatusUnauthorized, service.ErrUserNotLogin)
			return
		}
		operator, err := model.GetOperatorByCellphone(string(decriptBytes))
		if err != nil {
			getErrorResponse(c, http.StatusUnauthorized, service.ErrUserNotLogin)
			return
		}
		c.Set("op", operator)
		c.Next()
		return
	}
}

func getErrorResponse(c *gin.Context, httpStatus int, err *service.Error) {
	c.JSON(httpStatus, gin.H{
		"code": httpStatus,
		"data": 0,
		"msg":  err.Message(),
	})
	c.Abort()
}

// CustomerInfo 从请求中获取 customer 信息
func CustomerInfo(c *gin.Context) (*model.KroCustomer, error) {
	op, ok := c.Get("customer")
	if !ok {
		return nil, service.ErrUnauthorized
	}

	customer, ok := op.(*model.KroCustomer)
	if !ok {
		return nil, service.ErrUnauthorized
	}
	return customer, nil
}

// OperatorInfo 从请求中获取 operator 信息
func OperatorInfo(c *gin.Context) (*model.KroOperator, error) {
	op, ok := c.Get("op")
	if !ok {
		return nil, service.ErrUnauthorized
	}

	operator, ok := op.(*model.KroOperator)
	if !ok {
		return nil, service.ErrUnauthorized
	}
	return operator, nil
}
