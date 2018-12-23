package model

import (
	"sync"
	"time"

	"code.byted.org/gopkg/logs"
)

type SmsMsg struct {
	CustomerID int       `gorm:"column:customer_id"`
	Cellphone  string    `gorm:"column:cellphone"`
	Code       string    `gorm:"column:code"`
	SendTime   time.Time `gorm:"column:send_time"`
}

type SmsMsgDao struct{}

var smsMsgDao *SmsMsgDao
var smsMsgDaoOnce sync.Once

func SmsMsgDaoInstance() *SmsMsgDao {
	smsMsgDaoOnce.Do(
		func() {
			smsMsgDao = &SmsMsgDao{}
		})
	return smsMsgDao
}

func (dao *SmsMsgDao) GetPhoneLatestSms(phone string) (*SmsMsg, error) {
	var msg SmsMsg
	err := MSDB.Where("cellphone=?", phone).Order("id desc").First(&msg).Error
	return &msg, err
}

func (dao *SmsMsgDao) AddSmsMsg(smsMsg *SmsMsg) error {
	err := MSDB.Create(smsMsg).Error
	if err != nil {
		logs.Error("create smsMsg error, err=%+v", err)
	}
	return err
}
