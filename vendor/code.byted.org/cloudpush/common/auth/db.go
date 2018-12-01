package auth

import (
	"fmt"
	"time"

	"code.byted.org/golf/ssconf"
	"code.byted.org/gopkg/dbutil/conf"
	"code.byted.org/gopkg/dbutil/gormdb"
	"code.byted.org/gopkg/gorm"
)

var dboptional *conf.DBOptional

type PushUser struct {
	Id         int64
	Name       string `sql:"size:64"`
	Ak         string `sql:"size:64"`
	Sk         string `sql:"size:64"`
	Creator    string
	Owner      string
	Status     int32
	IpList     string //`sql:"size:2048"`
	AmsTag     string //`sql:"size:128"`
	Privilege  string //`sql:"size:2048"`
	ModifyTime time.Time
	CreateTime time.Time
}

func SetDBConf(optional *conf.DBOptional) {
	dboptional = optional
}

func InitDBConf(configFile string) {
	ssConf, _ := ssconf.LoadSsConfFile(configFile)
	dboptional_tmp := conf.GetDbConf(ssConf, "pushauthdb", conf.Read)
	dboptional = &dboptional_tmp
}

func GetUsersFromDB() ([]PushUser, error) {
	if dboptional == nil {
		return nil, fmt.Errorf("db not config")
	}
	handler := gormdb.NewDBHandler()

	var err error
	var db *gorm.DB
	err = handler.ConnectDB(dboptional)
	if err != nil {
		return nil, fmt.Errorf("Conntect DB Failed: %v", err)
	}

	db, err = handler.GetConnection()
	if err != nil {
		return nil, fmt.Errorf("Get DB Conntection Failed: %v", err)
	}

	var users []PushUser
	db.Where("status = 1").Find(&users)

	err = handler.Close()
	if err != nil {
		return nil, fmt.Errorf("Close DB Failed")
	}

	//20170808，由于mysql库的问题，在出现超时等情况，可能出现err为nil，但返回数据为空，而导致上游出错
	//所以添加此判断，防止本地文件及内存数据被空数据给替换掉
	if len(users) == 0 {
		return nil, fmt.Errorf("Select DB ,get no data!!!!!!")
	}

	return users, nil
}
