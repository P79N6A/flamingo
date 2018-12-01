package service

import (
	"code.bean.com/flamingo/model"
	"code.byted.org/gopkg/logs"
)

type ConfigService struct {
	configItemDao *model.ConfigItemDao
}

func NewConfigService() *ConfigService {
	return &ConfigService{
		configItemDao: model.ConfigItemDaoInstance(),
	}
}

func (service *ConfigService) GetAccessToken() string {
	token, err := service.configItemDao.GetConfigItem("access_token")
	if err != nil {
		logs.Error("get access token error:%+v", err)
		return ""
	}
	return token
}

func (service *ConfigService) UpdateAccessToken(value string) {
	service.configItemDao.UpdateConfigItem("access_token", value)
}
