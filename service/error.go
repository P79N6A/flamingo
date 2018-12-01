package service

import (
	"encoding/json"
	"net/http"
)

// 错误码列表
const (
	OK                          = 0 // 成功
	StatusMissParam             = 1 //缺少必要参数
	StatusInvalidParam          = 2 //参数值错误
	StatusEncryptionError       = 3 //加密错误
	StatusInternalServerError   = 4 //系统内部错误
	StatusDeviceAntiFraudExpire = 5 // 设备反欺诈信息过期
	StatusUserAlreadyExist      = 6 // 用户信息以存在
	StatusInvalidFourElements   = 7 //用户四要素非法
	StatusUserLoanReqError      = 8 //用户借款提交失败 （合作方）
	StatusUserRepayError        = 9 // 用户还款提交失败（合作方）
)

var errorMsgs map[int]string

// InitError 错误消息初始化
func InitError() {
	errorMsgs = make(map[int]string)
	//TODO 补充具体的错误消息
	errorMsgs[StatusMissParam] = "缺少必要参数"
	errorMsgs[StatusInvalidParam] = "参数值不合法"
	errorMsgs[StatusEncryptionError] = "加密结果错误"
	errorMsgs[StatusInternalServerError] = "服务器开小差啦"
	errorMsgs[StatusDeviceAntiFraudExpire] = "设备反欺诈信息过期"
	errorMsgs[StatusUserAlreadyExist] = "用户信息已存在"
	errorMsgs[StatusInvalidFourElements] = "非法的四要素值"
}

// Error 自定义错误类型
type Error struct {
	Code int
	Msg  string
	err  error
}

// Message 返回错误信息
func (e *Error) Message() string {
	if len(e.Msg) > 0 {
		return e.Msg
	}
	//TODO 根据code返回对于错误消息
	msg, ok := errorMsgs[e.Code]
	if ok {
		e.Msg = msg
	}
	return ""
}

// MarshalJSON error to json
func (e *Error) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"code": e.Code,
		"msg":  e.Message(),
	}
	if e.err != nil {
		m["detail"] = e.err.Error()
	}
	return json.Marshal(m)
}

// Error 返回错误信息
func (e *Error) Error() string {
	buf, _ := e.MarshalJSON()
	return string(buf)
}

// NewError 自定义一个错误
func NewError(code int, msg string) *Error {
	return &Error{
		Code: code,
		Msg:  msg,
	}
}

// ErrorWrap 创建一个错误
func ErrorWrap(code int, err error) *Error {
	return &Error{
		Code: code,
		err:  err,
	}
}

var (
	ErrUnauthorized = NewError(http.StatusUnauthorized, "Unauthorized")

	ErrMissParam         = NewError(4100, "缺少参数")
	ErrInvalidParam      = NewError(4101, "参数无效")
	ErrMissingData       = NewError(4102, "数据丢失")
	ErrUserNotLogin      = NewError(4103, "用户未登录")
	ErrIllegalDataAccess = NewError(4104, "非法数据访问")

	// 密码相关 42xx 开头
	ErrWrongPassword             = NewError(4200, "密码错误")
	ErrPasswordAlreadySet        = NewError(4201, "已设置过密码")
	ErrPasswordCheckCodeNotMatch = NewError(4202, "验证码错误")

	ErrorUserNotFound     = NewError(4301, "用户信息不存在")
	ErrorUserAlreadyExist = NewError(4302, "用户信息已存在")

	ErrorServiceInternalError = NewError(5001, "服务异常，请稍后再试")
)
