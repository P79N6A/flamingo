package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"sync"
	"time"

	"code.byted.org/gopkg/logs"
	"github.com/jinzhu/gorm"

	"code.bean.com/flamingo/config"
	"code.bean.com/flamingo/model"
	"code.bean.com/flamingo/util"

	"code.bean.com/flamingo/handler/view"
)

type CustomerService struct{}

var customerService *CustomerService
var customerServiceOnce sync.Once

var appID string
var appKey string

var smsURL = "https://yun.tim.qq.com/v5/tlssmssvr/sendsms?sdkappid=%s&random=%s"
var sigTpl = "appkey=%s&random=%s&time=%d&mobile=%s"

type CheckCodeReqTemplate struct {
	Params    []string `json:"params"`
	Sig       string   `json:"sig"`
	Tel       *TelInfo `json:"tel"`
	TimeStamp int64    `json:"time"`
	TplID     int      `json:"tpl_id"`
}
type CheckCodeResponse struct {
	Result int    `json:"result"`
	ErrMsg string `json:"errmsg"`
}
type TelInfo struct {
	Mobile     string `json:"mobile"`
	NationCode string `json:"nationcode"`
}

func CustomerServiceInstance() *CustomerService {
	customerServiceOnce.Do(
		func() {
			customerService = &CustomerService{}
			appID, _ = config.ConfigJson.Get("app_id").String()
			appKey, _ = config.ConfigJson.Get("app_key").String()
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
	if IsInvalidPhoneNo(phone) {
		logs.Error("invalid phone no:%s", phone)
		return nil, ErrIllegalPhoneNo
	}
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

func (s *CustomerService) SendCheckCode(phone string) error {
	if IsInvalidPhoneNo(phone) {
		logs.Error("invalid phone no:%s", phone)
		return ErrIllegalPhoneNo
	}
	customer, err := model.CustomerDaoInstance().GetCustomerByCellphone(phone)
	if err == gorm.ErrRecordNotFound {
		logs.Error("customer not found")
		return ErrIllegalPhoneNo
	}
	msg, err := model.SmsMsgDaoInstance().GetPhoneLatestSms(phone)
	if err != nil && err != gorm.ErrRecordNotFound {
		logs.Error("get phone latest")
		return ErrorServiceInternalError
	}
	if err == nil {
		if msg.SendTime.Add(2 * time.Minute).After(time.Now()) {
			return ErrIllegalDataAccess
		}
	}
	code, err := sendSMSMsg(phone)
	if err != nil {
		return ErrorServiceInternalError
	}
	return model.SmsMsgDaoInstance().AddSmsMsg(&model.SmsMsg{CustomerID: customer.ID, Cellphone: phone, Code: code, SendTime: time.Now()})
}

func (s *CustomerService) VerifyCheckCode(code, phone string) (bool, error) {
	if IsInvalidPhoneNo(phone) {
		logs.Error("invalid phone no:%s", phone)
		return false, ErrIllegalPhoneNo
	}
	_, err := model.CustomerDaoInstance().GetCustomerByCellphone(phone)
	if err == gorm.ErrRecordNotFound {
		return false, ErrIllegalPhoneNo
	}
	msg, err := model.SmsMsgDaoInstance().GetPhoneLatestSms(phone)
	if err != nil {
		return false, ErrorServiceInternalError
	}
	// if msg.SendTime.Add(3 * time.Minute).Before(time.Now()) {
	// 	return false, nil
	// }
	return msg.Code == code, nil
}

func IsInvalidPhoneNo(phone string) bool {
	reg := `^1([38][0-9]|14[57]|5[^4])\d{8}$`
	rgx := regexp.MustCompile(reg)
	return !rgx.MatchString(phone)
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

// CreateCaptcha 生成随机数
func CreateCaptcha() string {
	return fmt.Sprintf("%04v", rand.New(rand.NewSource(time.Now().UnixNano())).Int31n(10000))
}

func sendSMSMsg(phone string) (string, error) {
	code := CreateCaptcha()
	telInfo := &TelInfo{Mobile: phone, NationCode: "86"}
	params := []string{code}

	timeStamp := time.Now().Unix()
	url := fmt.Sprintf(smsURL, appID, code)

	sigInput := fmt.Sprintf(sigTpl, appKey, code, timeStamp, phone)
	logs.Info("input%s", sigInput)
	h := sha256.New()
	h.Write([]byte(sigInput))
	bs := h.Sum(nil)
	sig := fmt.Sprintf("%x", bs)
	reqInfo := CheckCodeReqTemplate{Params: params, Sig: sig, Tel: telInfo, TimeStamp: timeStamp, TplID: 253094}
	var resp CheckCodeResponse
	logs.Info("send sms req:%+v", reqInfo)
	err := util.PostWithObjResponse(context.Background(), url, reqInfo, &resp)
	if err != nil {
		return "", err
	}
	if resp.Result == 0 {
		return code, nil
	}
	return "", ErrorServiceInternalError
}
