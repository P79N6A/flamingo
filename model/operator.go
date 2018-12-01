package model

import (
	"code.byted.org/gopkg/logs"
)

type KroOperator struct {
	ID        int    `gorm:"column:id" json:"-"`
	Cellphone string `gorm:"column:cellphone" json:"cellphone"`
	Name      string `gorm:"column:name" json:"name"`
	Pwd       string `gorm:"column:pwd" json:"-"`
}

func CheckOperatorByPwd(cellphone, pwd string) (*KroOperator, error) {
	var operator KroOperator
	err := MSDB.Where("cellphone=? AND pwd=?", cellphone, pwd).First(&operator).Error
	if err != nil {
		logs.Error("get customer error, err=%+v", err)
		return nil, err
	}

	return &operator, nil
}

func GetOperatorByCellphone(cellphone string) (*KroOperator, error) {
	var operator KroOperator
	err := MSDB.Where("cellphone=?", cellphone).First(&operator).Error
	if err != nil {
		logs.Error("get customer error, err=%+v", err)
		return nil, err
	}

	return &operator, nil
}
