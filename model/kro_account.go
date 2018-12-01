package model

import (
	"sync"
	"time"

	"code.byted.org/gopkg/logs"
)

const (
	AccountTypeRecharge = "RECHARGE" //充值
	AccountTypeCunsume  = "CONSUME"  //消费
	AcccountTypeRefund  = "REFUND"   //退款
)

type KroAccount struct {
	CustomerID  int       `gorm:"column:customer_id"`
	AccountType string    `gorm:"column:account_type"`
	Amount      int       `gorm:"column:amount"`
	DealTime    time.Time `gorm:"column:deal_time"`
	Desc        string    `gorm:"column:desc"`
	OpCell      string    `gorm:"column:operator"`
	Operator    string    `gorm:"column:operator_name"`
}

type KroAccountDao struct{}

var kroAccountDao *KroAccountDao
var kroAccountDaoOnce sync.Once

func KroAccountDaoInstance() *KroAccountDao {
	kroAccountDaoOnce.Do(
		func() {
			kroAccountDao = &KroAccountDao{}
		})
	return kroAccountDao
}

func (dao *KroAccountDao) CreateNewAccount(account *KroAccount) error {
	err := MSDB.Create(account).Error
	if err != nil {
		logs.Error("create account error, err=%+v", err)
	}
	return err
}

func (dao *KroAccountDao) GetCustomerAccounts(customer *KroCustomer) ([]*KroAccount, error) {
	accounts := make([]*KroAccount, 0)
	err := MSDB.Where("customer_id=?", customer.ID).Order("id desc").Find(&accounts).Error
	if err != nil {
		logs.Error("get customer accounts error, err=%+v", err)
	}
	return accounts, err
}
