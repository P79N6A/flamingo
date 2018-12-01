package model

import (
	"errors"
	"sync"
	"time"

	"code.byted.org/gopkg/logs"
	"github.com/jinzhu/gorm"
)

type KroCustomer struct {
	ID         int       `gorm:"column:id"`
	CustomerID string    `gorm:"column:card_no"`
	Name       string    `gorm:"column:name"`
	Cellphone  string    `gorm:"column:cellphone"`
	OpenDate   time.Time `gorm:"column:open_date"`
}

type KroCustomerDao struct{}

var kroCustomerDao *KroCustomerDao
var kroCustomerDaoOnce sync.Once

func CustomerDaoInstance() *KroCustomerDao {
	kroCustomerDaoOnce.Do(
		func() {
			kroCustomerDao = &KroCustomerDao{}
		})
	return kroCustomerDao
}

func (dao *KroCustomerDao) CreateCustomer(customer *KroCustomer) error {
	err := MSDB.Where("cellphone=?", customer.Cellphone).First(&KroCustomer{}).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		logs.Error("get customer error, err=%+v", err)
		return err
	}
	if err == nil {
		return errors.New("user exist")
	}
	err = MSDB.Create(customer).Error
	if err != nil {
		logs.Error("create customer error, err=%+v", err)
	}
	return err
}

func (dao *KroCustomerDao) GetCustomerByCellphone(cellphone string) (*KroCustomer, error) {
	var customer KroCustomer
	err := MSDB.Where("cellphone=?", cellphone).First(&customer).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		logs.Error("get customer error, err=%+v", err)
	}
	return &customer, err
}
