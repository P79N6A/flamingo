package model

import (
	"sync"
)

var (
	configItemDao     *ConfigItemDao
	configItemDaoOnce sync.Once
)

type ConfigItemDao struct{}

type ConfigItem struct {
	ID    int    `gorm:"column:id"`
	Key   string `gorm:"column:key_field"`
	Value string `gorm:"column:value_field"`
}

func ConfigItemDaoInstance() *ConfigItemDao {
	configItemDaoOnce.Do(func() {
		configItemDao = &ConfigItemDao{}
	})
	return configItemDao
}

func (dao *ConfigItemDao) GetConfigItem(key string) (string, error) {
	var item ConfigItem
	err := MSDB.Where("key_field = ?", key).First(&item).Error
	if err != nil {
		return "", err
	}
	return item.Value, nil
}

func (dao *ConfigItemDao) UpdateConfigItem(key, value string) {
	var item ConfigItem
	MSDB.Model(&item).Where("key_field = ?", key).Update("value_field", value)
}
