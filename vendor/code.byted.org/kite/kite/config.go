package kite

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"code.byted.org/bagent/go-client"
	"code.byted.org/gopkg/env"
	"code.byted.org/gopkg/logs"
	"code.byted.org/gopkg/thrift"
	"code.byted.org/kite/kitc"
	"code.byted.org/kite/kitenv"
	"gopkg.in/yaml.v2"
)

const (
	DefaultMaxConns              int64 = 10000
	DefaultLimitQps              int64 = 50000
	DefaultTransportBufferedSize int   = 4096

	_ENV_CONFIG_FILE  = "KITE_CONFIG_FILE"
	_ENV_LOG_DIR      = "KITE_LOG_DIR"
	_ENV_SERVICE_NAME = "KITE_SERVICE_NAME"
	_ENV_SERVICE_PORT = "KITE_SERVICE_PORT"
	_ENV_DEBUG_PORT   = "KITE_DEBUG_PORT"

	_TCE_SERVICE_PORT = "RUNTIME_SERVICE_PORT"
)

var (
	// 全局的RpcServer实例
	RpcService *RpcServer
	// Thrift Processor
	Processor     thrift.TProcessor
	ServiceConfig ConfigInterface
	// 服务名称 psm
	ServiceName string
	// 当前服务版本号，由commit ID的前缀和编译时间组成, 编译的时候打入
	ServiceVersion string = "DefaultVersion"
	// RPC配置文件目录，渐渐会废弃掉
	RpcConfDir string
	// 启动时间
	StartTime time.Time
	// 服务当前所在集群
	ServiceCluster string

	// 服务IP地址
	ServiceAddr string
	// 服务端口
	ServicePort string

	ReadWriteTimeout time.Duration

	EnableMonitor     bool
	MonitorHostPort   string
	EnableDebugServer bool
	EnableMetrics     bool
	DebugServerPort   string
	ServicePath       string
	ConfigFile        string
	ConfigDir         string
	ConfigEnv         string
	LogDir            string
	LogLevel          int

	LogFile     string
	MaxLogSize  int64
	LogInterval string

	// log provider
	ConsoleLog bool
	ScribeLog  bool
	FileLog    bool
	DatabusLog bool

	LocalIp string // machine IP

	// bagent
	BagentClient *bagentutil.Client
)

func Usage() {
	usage := `
	-conf  config file
	-log   log dir
	-svc   svc name
	-port  listen port
	-loanrpc   loanrpc conf dir
	`
	fmt.Fprintln(os.Stderr, os.Args[0], usage)
	os.Exit(-1)
}

// Init .
func Init() {
	// in tce env, MY_CPU_LIMIT will be set as limit cpu cores.
	if v := os.Getenv("GOMAXPROCS"); v == "" {
		if v := os.Getenv("MY_CPU_LIMIT"); v != "" {
			n, err := strconv.ParseInt(v, 10, 64)
			if err == nil {
				runtime.GOMAXPROCS(int(n))
			}
		}
	}

	// ConfigFile = os.Getenv(_ENV_CONFIG_FILE)
	if ConfigFile == "" {
		flag.StringVar(&ConfigFile, "conf", "", "support config file.")
	}

	// LogDir = os.Getenv(_ENV_LOG_DIR)
	if LogDir == "" {
		flag.StringVar(&LogDir, "log", "", "support log dir.")
	}
	if ServiceName == "" {
		flag.StringVar(&ServiceName, "svc", "", "support svc name.")
	}
	if ServicePort == "" {
		flag.StringVar(&ServicePort, "port", "", "support service port")
	}

	if RpcConfDir == "" {
		flag.StringVar(&RpcConfDir, "loanrpc", "", "support loanrpc conf dir")
	}
	flag.Parse()
	ConfigDir = path.Dir(ConfigFile)
	ConfigEnv = os.Getenv("CONF_ENV")

	// 用环境变量中的值覆盖命令行参数传入
	if v := os.Getenv(_ENV_SERVICE_NAME); v != "" {
		ServiceName = v
	}
	if v := os.Getenv(_ENV_SERVICE_PORT); v != "" {
		ServicePort = v
	}
	// Use TCE PORT instead
	if v := os.Getenv(_TCE_SERVICE_PORT); v != "" {
		ServicePort = v
	}

	ServiceCluster = os.Getenv("SERVICE_CLUSTER")

	if ConfigFile == "" {
		fmt.Fprintf(os.Stderr, "configfile is empty, use -conf option or %s environment", _ENV_CONFIG_FILE)
		Usage()
	}

	if LogDir == "" {
		fmt.Fprintf(os.Stderr, "logdir is empty, use -log option or %s environment", _ENV_LOG_DIR)
		Usage()
	}

	if ServiceName == "" {
		fmt.Fprintf(os.Stderr, "servicename is empty, use -svc option or %s environment", _ENV_SERVICE_NAME)
		Usage()
	}

	var err error
	ConfigFile, err = filepath.Abs(ConfigFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Get abs config file error: %s\n", err)
		os.Exit(-1)
	}

	LogDir, err = filepath.Abs(LogDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Get abs log dir error: %s\n", err)
		os.Exit(-1)
	}
	LogFile = filepath.Join(LogDir, "app", ServiceName+".log")

	LocalIp = GetLocalIp()

	InitDatabusChannel()
	ParseConfig()

	if ServicePort == "" {
		fmt.Fprintln(os.Stderr, "support service port.")
		Usage()
	}

	if !EnableMetrics {
		metricsClient = &EmptyEmiter{}
	}

	if ConsoleLog {
		InitConsole()
	}
	AccessLogger = NewAccessLogger(filepath.Join(LogDir, "app"), ServiceName, true)
	CallLogger = NewCallLogger(filepath.Join(LogDir, "loanrpc"), ServiceName, true)
	kitc.SetCallLog(CallLogger)
	if FileLog {
		switch LogInterval {
		case "day":
			InitFile(LogLevel, LogFile, logs.HourDur, MaxLogSize)
		case "hour":
			InitFile(LogLevel, LogFile, logs.HourDur, MaxLogSize)
		default:
			InitFile(LogLevel, LogFile, logs.HourDur, MaxLogSize)
		}
	}
	InitLog(LogLevel)
	InitDatabus(LogLevel)

	RpcService = NewRpcServer()

	BagentClient, err = bagentutil.NewClient()
	if err == nil {
		ReportMetadata()
	} else {
		fmt.Fprintln(os.Stderr, "Bagent err: ", err)
	}

	kitc.SetKiteService(ServiceName, ServiceCluster)
	kitc.SetReporter(newKiteReporter())
}

// InitDatabusChannel .
func InitDatabusChannel() {
	switch env.IDC() {
	case env.DC_LF, env.DC_HY:
		databusAPPChannel = "__LOG__"
		databusRPCChannel = "web_rpc_log"
		// support test env
		testPrefix := os.Getenv("TESTING_PREFIX")
		if testPrefix != "" {
			databusAPPChannel = testPrefix + "_" + "normal_log"
			databusRPCChannel = testPrefix + "_" + databusRPCChannel
		}
	case env.DC_SG, env.DC_VA:
		databusAPPChannel = "i18n__LOG__main"
		databusRPCChannel = "i18n_web_rpc_log_main"
	}
}

// ParseConfig loads kite's config
func ParseConfig() {
	var err error
	cfg, err := NewYamlFromFile(ConfigFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can not parse config file %s\n", ConfigFile)
		os.Exit(-1)
	}

	if os.Getenv("IS_PROD_RUNTIME") != "" {
		ServiceConfig = GetConfigItem(cfg, "Product")
	} else {
		if kitenv.Product() {
			ServiceConfig = GetConfigItem(cfg, "Product")
		} else {
			ServiceConfig = GetConfigItem(cfg, "Develop")
		}
	}

	if ServicePort == "" {
		ServicePort = ServiceConfig.DefaultString("ServicePort", ServicePort)
	}

	// ServiceAddr
	// EnableMetrics
	// EnableDebugServer
	// ConsoleLog
	// ScribeLog
	// FileLog
	// DebugServerPort
	// LogLevel
	// MaxLogSize
	// LogInterval
	// LimitQPS
	// LimitConnection
	// BagentAddr
	// AccessControl
	ServiceAddr = ServiceConfig.DefaultString("ServiceAddr", ":")
	EnableMetrics = ServiceConfig.DefaultBool("EnableMetrics", false)
	EnableDebugServer = ServiceConfig.DefaultBool("EnableDebugServer", true)

	ConsoleLog = ServiceConfig.DefaultBool("ConsoleLog", true)
	ScribeLog = ServiceConfig.DefaultBool("ScribeLog", false)
	FileLog = ServiceConfig.DefaultBool("FileLog", false)
	DatabusLog = ServiceConfig.DefaultBool("DatabusLog", true) // 默认向Databus打

	DebugServerPort = ServiceConfig.DefaultString("DebugServerPort", ":1"+ServicePort) // 默认pprof地址
	if debugPort := os.Getenv(_ENV_DEBUG_PORT); len(debugPort) != 0 {
		DebugServerPort = ":" + debugPort
	}
	LogLevel = ServiceConfig.DefaultInt("LogLevel", 0)                    // 默认使用Trace等级
	MaxLogSize = ServiceConfig.DefaultInt64("MaxLogSize", 1024*1024*1024) // 1G
	LogInterval = ServiceConfig.DefaultString("LogInterval", "day")       // 默认按天切分

	duration := ServiceConfig.DefaultString("ReadWriteTimeout", "3s")
	ReadWriteTimeout, err = time.ParseDuration(duration)
	if err != nil {
		ReadWriteTimeout, _ = time.ParseDuration("3s")
	}

	limitQPS = ServiceConfig.DefaultInt64("LimitQPS", DefaultLimitQps)
	limitMaxConns = ServiceConfig.DefaultInt64("LimitConnections", DefaultMaxConns)

}

// DefineProcessor
func DefineProcessor(p thrift.TProcessor) {
	if Processor != nil {
		panic("DefineProcessor more than onece")
	}
	Processor = p
}

// UnmarshallYMLConfig parses the file specified by confName and CONF_ENV to a YML object;
// If file "confName.CONF_ENV" doesn't exist, then try to unmarshal "confName" as default;
func UnmarshallYMLConfig(confName string, obj interface{}) error {
	buf, err := ReadConfig(confName)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(buf, obj)
}

// UnmarshallYMLConfigWithEnv parses the file specified by confName and confEnv to a YML object
func UnmarshallYMLConfigWithEnv(confName, confEnv string, obj interface{}) error {
	buf, err := ReadConfigWithEnv(confName, confEnv)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(buf, obj)
}

// ReadConfig reads the config specified by confName and CONF_ENV;
// If file "confName.CONF_ENV" doesn't exist, then try to read "confName" as default;
func ReadConfig(confName string) ([]byte, error) {
	buf, err := ReadConfigWithEnv(confName, GetConfEnv())
	if err == nil || os.IsNotExist(err) == false {
		return buf, err
	}

	filePath := path.Join(ConfigDir, confName)
	logs.Warnf("use %v as default config", filePath)
	return ioutil.ReadFile(filePath)
}

// ReadConfigWithEnv .
func ReadConfigWithEnv(confName, confEnv string) ([]byte, error) {
	ext := strings.TrimPrefix(confEnv, ".")
	base := path.Join(ConfigDir, confName)
	if ext != "" {
		base = fmt.Sprintf("%v.%v", base, ext)
	}
	return ioutil.ReadFile(base)
}

// GetConfEnv returns the CONF_ENV
func GetConfEnv() string {
	return ConfigEnv
}

// GetConfDir returns where configs are
func GetConfDir() string {
	return ConfigDir
}
