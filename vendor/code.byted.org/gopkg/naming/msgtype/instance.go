package msgtype

import "strconv"

//go:generate msgp
type Instance struct {
	Addr      string `msg:"a" json:"a"`
	Cluster   string `msg:"c" json:"c"`
	Weight    uint16 `msg:"w" json:"w"`
	UpdatedAt int64  `msg:"u" json:"u"`
}

//go:generate msgp
type ServiceInstaces struct {
	RequestID string     `msg:"rid" json:"rid"`
	Service   string     `msg:"s" json:"s"`
	Total     int64      `msg:"n" json:"n"`
	Instances []Instance `msg:"ii" json:"ii"`
}

func (i Instance) String() string {
	return "[a:" + i.Addr + " c:" + i.Cluster + " w:" + strconv.Itoa(int(i.Weight)) + "]"
}

func (ii ServiceInstaces) Addrs() []string {
	ret := make([]string, len(ii.Instances))
	for i, e := range ii.Instances {
		ret[i] = e.Addr
	}
	return ret
}
