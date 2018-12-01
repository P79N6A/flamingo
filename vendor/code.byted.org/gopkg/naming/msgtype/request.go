package msgtype

const (
	OpQuery = 1
)

//go:generate msgp
type Request struct {
	RequestID  string `msg:"rid" json:"rid"`
	Op         uint8  `msg:"op" json:"op"`
	Service    string `msg:"s" json:"s"`
	Cluster    string `msg:"c" json:"c"`
	NoCompress bool   `msg:"nc" json:"nc"`
	SingleShot bool   `msg:"ss" json:"ss"`
	Limit      uint16 `msg:"l" json:"l"`
}
