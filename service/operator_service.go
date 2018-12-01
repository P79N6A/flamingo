package service

import (
	"sync"

	"code.bean.com/flamingo/model"
)

type OperatorSerivce struct{}

var operatorService *OperatorSerivce
var operatorOnce sync.Once

func OperatorServiceInstance() *OperatorSerivce {
	operatorOnce.Do(func() {
		operatorService = &OperatorSerivce{}
	})
	return operatorService
}

func (s *OperatorSerivce) OperatorLogin(cell, pwd string) (*model.KroOperator, error) {
	operator, err := model.CheckOperatorByPwd(cell, pwd)
	return operator, err
}
