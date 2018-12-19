package service

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"code.byted.org/gopkg/logs"
	"github.com/jinzhu/gorm"

	"code.bean.com/flamingo/model"

	"code.bean.com/flamingo/handler/view"
)

type CustomerService struct{}

var customerService *CustomerService
var customerServiceOnce sync.Once

func CustomerServiceInstance() *CustomerService {
	customerServiceOnce.Do(
		func() {
			customerService = &CustomerService{}
		})
	return customerService
}

func (s *CustomerService) GetCustomerDetailInfo(cellphone string) (*view.CustomersInfo, error) {
	customer, err := model.CustomerDaoInstance().GetCustomerByCellphone(cellphone)
	if err == gorm.ErrRecordNotFound {
		return nil, ErrorUserNotFound
	}
	if err != nil {
		logs.Error("get customer info failed,err=%+v", err)
		return nil, err
	}
	accounts, err := model.KroAccountDaoInstance().GetCustomerAccounts(customer)
	if err != nil && err != gorm.ErrRecordNotFound {
		logs.Error("get customer accounts error, err=%+v", err)
		return nil, err
	}
	logs.Info("Accounts:%d", len(accounts))
	accountInfos := make([]*view.AccountInfo, 0)
	restAmount := 0
	for _, account := range accounts {
		amount := fmt.Sprintf("%.2f", float64(account.Amount)/100.00)
		dealTime := account.DealTime.Format("2006-01-02 15:04:05")
		accountType := GetAccountType(account.AccountType)
		accountInfos = append(accountInfos, &view.AccountInfo{
			AccountAmount: amount,
			AccountTime:   dealTime,
			AccountType:   accountType,
			OperatorName:  account.Operator,
		})
		if account.AccountType == model.AccountTypeRecharge || account.AccountType == model.AcccountTypeRefund {
			logs.Info("amount:%d", account.Amount)
			restAmount = restAmount + account.Amount
		} else {
			logs.Info("ccamount:%d", account.Amount)
			restAmount = restAmount - account.Amount
		}
	}
	rest := fmt.Sprintf("%.2f", float64(restAmount)/100.00)
	return &view.CustomersInfo{
		CustomerCellphone:  customer.Cellphone,
		CustomerName:       customer.Name,
		CustomerRestAmount: rest,
		CustomerOpenDate:   customer.OpenDate.Format("2006-01-02 15:04:05"),
		AccountsDetail:     accountInfos,
	}, nil
}

func (s *CustomerService) CreateCustomer(phone, name string) (*view.CustomersInfo, error) {
	customer, err := model.CustomerDaoInstance().GetCustomerByCellphone(phone)
	if err != nil && err != gorm.ErrRecordNotFound {
		logs.Error("get customer info failed,err=%+v", err)
		return nil, err
	}
	if customer.ID > 0 {
		return nil, ErrorUserAlreadyExist
	}
	customerID := IDGenerator()
	customer = &model.KroCustomer{CustomerID: customerID, Name: name, Cellphone: phone, OpenDate: time.Now()}
	err = model.CustomerDaoInstance().CreateCustomer(customer)
	if err != nil {
		logs.Error("create new customer error, err=%+v", err)
		return nil, err
	}
	return &view.CustomersInfo{
		CustomerOpenDate: customer.OpenDate.Format("2006-01-02 15:04:05"),
		AccountsDetail:   make([]*view.AccountInfo, 0),
	}, nil
}

func (s *CustomerService) AddCustomerAccount(phone, operate, amount, desc string, operator *model.KroOperator) (bool, error) {
	customer, err := model.CustomerDaoInstance().GetCustomerByCellphone(phone)
	if err != nil {
		logs.Error("get customer info failed,err=%+v", err)
		return false, err
	}
	fAmount, err := strconv.ParseFloat(amount, 10)
	if err != nil {
		logs.Error("parse amount to num error,err=%+v", err)
		return false, err
	}
	if operate == model.AccountTypeCunsume {
		detailInfo, _ := s.GetCustomerDetailInfo(phone)
		restAmount, _ := strconv.ParseFloat(detailInfo.CustomerRestAmount, 10)
		if fAmount > restAmount {
			return false, NewError(401, "买单金额超出账户余额")
		}
	}
	err = model.KroAccountDaoInstance().CreateNewAccount(&model.KroAccount{CustomerID: customer.ID, AccountType: operate, Amount: int(fAmount * 100), DealTime: time.Now(), Desc: desc, OpCell: operator.Cellphone, Operator: operator.Name})
	if err != nil {
		logs.Error("create new account item error,err=%+v", err)
		return false, err
	}
	return true, nil
}

func IDGenerator() string {
	buf := make([]byte, 0, 64)
	buf = time.Now().AppendFormat(buf, "20060102150405")
	return string(buf)
}

func GetAccountType(accountType string) string {
	switch accountType {
	case model.AccountTypeCunsume:
		return "消费"
	case model.AccountTypeRecharge:
		return "充值"
	default:
		return "退款"
	}
}
