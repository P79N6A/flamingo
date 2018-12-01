package model

import (
	"fmt"

	"code.bean.com/flamingo/config"
	"code.byted.org/gopkg/logs"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

var (
	MSDB *gorm.DB
)

func Init() error {
	return initMYSQL()
}

func initMYSQL() error {
	username, _ := config.ConfigJson.Get("flamingo_db").Get("username").String()
	pwd, _ := config.ConfigJson.Get("flamingo_db").Get("password").String()
	host, _ := config.ConfigJson.Get("flamingo_db").Get("host").String()
	database, _ := config.ConfigJson.Get("flamingo_db").Get("database").String()
	dbCon := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8&parseTime=True&loc=Local", username, pwd, host, database)
	db, err := gorm.Open("mysql", dbCon)
	if err != nil {
		logs.Info("init mysql error:%+v", err)
		return err
	}
	MSDB = db
	logs.Info("connect mysql success!!")
	return nil
}
