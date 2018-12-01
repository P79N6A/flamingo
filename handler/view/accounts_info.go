package view

type CustomersInfo struct {
	AccountsDetail     []*AccountInfo `json:"account_detail"`
	CustomerCellphone  string         `json:"cellphone"`
	CustomerName       string         `json:"customer_name"`
	CustomerRestAmount string         `json:"rest_amount"`
	CustomerOpenDate   string         `json:"open_date"`
}

type AccountInfo struct {
	AccountTime   string `json:"account_time"`
	AccountType   string `json:"type"`
	AccountAmount string `json:"amount"`
	OperatorName  string `json:"operator"`
}
