package config

import (
	"encoding/json"
	"io/ioutil"
	"log"

	simplejson "github.com/bitly/go-simplejson"
)

// 系统运行环境
const (
	Prod = "prod"
	Dev  = "dev"
)

//KafkaConfig kafka config
type KafkaConfig struct {
	RealtimeLogKafkaTopic string `json:"realtime_log_kafka_topic"`
	ReqRespLogKafkaTopic  string `json:"req_resp_log_kafka_topic"`
	KafkaService          string `json:"kafka_service"`
}

// EncryptSettings 加密信息配置
type EncryptSettings struct {
	BusinessName string `json:"business_name"`
	Token        string `json:"token"`
}

// Config 配置信息
type Config struct {
	Env string `json:"env"`
}

var (
	// ConfigInstance 当前环境配置信息
	ConfigInstance *Config
	// origin File 将配置信息解成json
	ConfigJson *simplejson.Json
)

// Product 检查当前环境是否是线上环境
func (c *Config) Product() bool {
	return c.Env == Prod
}

// Init 初始化配置文件
func Init(file string) error {
	conf, err := loadConfig(file)
	if err != nil {
		log.Fatalf("init conf instance error:%+v\n", err)
		return err
	}
	ConfigInstance = conf

	js, err := loadDBSettings(file)
	if err != nil {
		log.Fatalf("init conf as DBSettings error: %+v\n", err)
		return err
	}
	ConfigJson = js //Need 解析原始信息，被应用灵活使用
	return nil
}

func loadDBSettings(file string) (*simplejson.Json, error) {
	body, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatalf("load config file %s as DBSettings error: %+v", file, err)
		return nil, err
	}
	return simplejson.NewJson(body)
}

func loadConfig(file string) (*Config, error) {
	confContent, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatalf("create new config error: %+v\n", err)
		return nil, err
	}
	var conf Config
	err = json.Unmarshal(confContent, &conf)
	if err != nil {
		log.Fatalf("json unmarshal error: %+v\n", err)
		return nil, err
	}
	return &conf, nil
}
