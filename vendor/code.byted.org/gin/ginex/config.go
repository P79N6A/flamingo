package ginex

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"code.byted.org/gin/ginex/internal"
	"code.byted.org/whale/ginentry"
	"github.com/spf13/viper"
)

const (
	_PRODUCT_MODE    = "Product"
	_DEVELOP_MODE    = "Develop"
	_SERVICE_PORT    = "ServicePort"
	_DEBUG_PORT      = "DebugPort"
	_ENABLE_PPROF    = "EnablePprof"
	_LOG_LEVEL       = "LogLevel"
	_LOG_INTERVAL    = "LogInterval"
	_ENABLE_METRICS  = "EnableMetrics"
	_CONSOLE_LOG     = "ConsoleLog"
	_DATABUS_LOG     = "DatabusLog"
	_FILE_LOG        = "FileLog"
	_MODE            = "Mode"
	_SERVICE_VERSION = "ServiceVersion"

	_ENV_PSM          = "PSM"
	_ENV_CONF_DIR     = "GIN_CONF_DIR"
	_ENV_LOG_DIR      = "GIN_LOG_DIR"
	_ENV_SERVICE_PORT = "RUNTIME_SERVICE_PORT"
	_ENV_DEBUG_PORT   = "RUNTIME_DEBUG_PORT"
	_ENV_HOST_NETWORK = "IS_HOST_NETWORK"
)

var (
	appConfig AppConfig
)

// 对应框架配置文件配置项
type YamlConfig struct {
	ServicePort         int
	DebugPort           int
	EnablePprof         bool
	LogLevel            string
	LogInterval         string
	EnableMetrics       bool
	ConsoleLog          bool
	DatabusLog          bool
	FileLog             bool
	Mode                string
	ServiceVersion      string
	EnableAntiCrawl     bool
	AntiCrawlPathsWhite []string
	AntiCrawlPathsBlack []string
}

// 对应框架命令行参数配置项
type FlagConfig struct {
	PSM     string
	ConfDir string
	LogDir  string
	Port    int
}

type AppConfig struct {
	FlagConfig
	YamlConfig
}

func GetAntiCrawlConfig() *ginentry.Config {
	return &ginentry.Config{
		EnableAntiCrawl:     appConfig.EnableAntiCrawl,
		AntiCrawlPathsWhite: appConfig.AntiCrawlPathsWhite,
		AntiCrawlPathsBlack: appConfig.AntiCrawlPathsBlack,
	}
}

// PSM return app's PSM
func PSM() string {
	return appConfig.PSM
}

// ConfDir returns the app's config directory. It's a good practice to put all configure files in such directory,
// then you can access config file by filepath.Join(ginex.ConfDir(), "your.conf")
func ConfDir() string {
	return appConfig.ConfDir
}

// LogDir returns app's log root directory
func LogDir() string {
	return appConfig.LogDir
}

// config优先级: flag > env > file > default
// env does not work now
func loadConf() {
	// define and parse flags
	parseFlags()

	parseConf()

	fmt.Fprintf(os.Stdout, "App config: %#v\n", appConfig)
}

func parseConf() {
	// parse app config
	v := viper.New()
	v.SetEnvPrefix("GIN")

	confFile := filepath.Join(ConfDir(), strings.Replace(PSM(), ".", "_", -1)+".yaml")
	v.SetConfigFile(confFile)
	if err := v.ReadInConfig(); err != nil {
		msg := fmt.Sprintf("Failed to load app config: %s, %s", confFile, err)
		fmt.Fprintf(os.Stderr, "%s\n", msg)
		panic(msg)
	}
	mode := _DEVELOP_MODE
	if Product() {
		mode = _PRODUCT_MODE
	}

	vv := v.Sub(mode)
	if vv == nil {
		msg := fmt.Sprintf("Failed to parse config sub module: %s", mode)
		fmt.Fprintf(os.Stderr, "%s\n", msg)
		panic(msg)
	} else {
		setDefault(vv)
	}

	yamlConfig := &appConfig.YamlConfig
	if err := vv.Unmarshal(yamlConfig); err != nil {
		msg := fmt.Sprintf("Failed to unmarshal app config: %s", err)
		fmt.Fprintf(os.Stderr, "%s\n", msg)
		panic(msg)
	}
	parseServicePorts()

}

// parseServicePorts handles port configs in environment, config file and flag
func parseServicePorts() {
	var err error
	servicePortValue := os.Getenv(_ENV_SERVICE_PORT)
	debugPortValue := os.Getenv(_ENV_DEBUG_PORT)
	var hostNetWork bool
	if v := os.Getenv(_ENV_HOST_NETWORK); v != "" {
		if hostNetWork, err = strconv.ParseBool(v); err != nil {
			msg := fmt.Sprintf("Failed to convert environment variable: %s, %s", _ENV_HOST_NETWORK, err)
			fmt.Fprintf(os.Stderr, "%s\n", msg)
			panic(msg)
		}
	}

	if hostNetWork {
		// host模式: 只能使用环境变量端口, 否则直接报错
		if port, err := strconv.Atoi(servicePortValue); err != nil {
			msg := fmt.Sprintf("Failed to convert environment variable: %s, %s", _ENV_SERVICE_PORT, err)
			fmt.Fprintf(os.Stderr, "%s\n", msg)
			panic(msg)
		} else {
			appConfig.ServicePort = port
		}

		if debugPortValue == "" {
			appConfig.DebugPort = 0
		} else {
			if port, err := strconv.Atoi(debugPortValue); err != nil {
				msg := fmt.Sprintf("Failed to convert environment variable: %s, %s", _ENV_DEBUG_PORT, err)
				fmt.Fprintf(os.Stderr, "%s\n", msg)
				panic(msg)
			} else {
				appConfig.DebugPort = port
			}
		}
	} else {
		// 非host模式: 如果环境变量指定的端口,使用环境变量的.否则使用配置文件的端口
		if servicePortValue != "" {
			if port, err := strconv.Atoi(servicePortValue); err != nil {
				msg := fmt.Sprintf("Failed to convert environment variable: %s, %s", _ENV_SERVICE_PORT, err)
				fmt.Fprintf(os.Stderr, "%s\n", msg)
				panic(msg)
			} else {
				appConfig.ServicePort = port
			}
		}
		if debugPortValue != "" {
			if port, err := strconv.Atoi(debugPortValue); err != nil {
				msg := fmt.Sprintf("Failed to convert environment variable: %s, %s", _ENV_DEBUG_PORT, err)
				fmt.Fprintf(os.Stderr, "%s\n", msg)
				panic(msg)
			} else {
				appConfig.DebugPort = port
			}
		}
	}

	// flag指定的port优先级最高
	if appConfig.Port != 0 {
		appConfig.ServicePort = appConfig.Port
	}
}

func setDefault(v *viper.Viper) {
	v.SetDefault(_SERVICE_PORT, "6789")
	v.SetDefault(_DEBUG_PORT, "6790")
	v.SetDefault(_ENABLE_PPROF, false)
	v.SetDefault(_LOG_LEVEL, "debug")
	v.SetDefault(_LOG_INTERVAL, "hour")
	v.SetDefault(_ENABLE_METRICS, false)
	v.SetDefault(_CONSOLE_LOG, true)
	v.SetDefault(_DATABUS_LOG, false)
	v.SetDefault(_FILE_LOG, true)
	v.SetDefault(_MODE, "debug")
	v.SetDefault(_SERVICE_VERSION, "0.1.0")
}

func parseFlags() {
	flag.StringVar(&appConfig.PSM, "psm", "", "psm")
	flag.StringVar(&appConfig.ConfDir, "conf-dir", "", "support config file.")
	flag.StringVar(&appConfig.LogDir, "log-dir", "", "log dir.")
	flag.IntVar(&appConfig.Port, "port", 0, "service port.")
	flag.Parse()

	if appConfig.PSM == "" {
		appConfig.PSM = os.Getenv(_ENV_PSM)
	}
	if appConfig.PSM == "" {
		fmt.Fprintf(os.Stderr, "PSM is not specified, use -psm option or %s environment\n", _ENV_PSM)
		usage()
	} else {
		os.Setenv(internal.GINEX_PSM, appConfig.PSM)
	}
	if appConfig.ConfDir == "" {
		appConfig.ConfDir = os.Getenv(_ENV_CONF_DIR)
	}
	if appConfig.ConfDir == "" {
		fmt.Fprintf(os.Stderr, "Conf dir is not specified, use -conf-dir option or %s environment\n", _ENV_CONF_DIR)
		usage()
	}
	if appConfig.LogDir == "" {
		appConfig.LogDir = os.Getenv(_ENV_LOG_DIR)
	}
	if appConfig.LogDir == "" {
		fmt.Fprintf(os.Stderr, "Log dir is not specified, use -log-dir option or %s environment\n", _ENV_LOG_DIR)
		usage()
	}
}

func usage() {
	flag.Usage()
	os.Exit(-1)
}
