package kite

import (
	"fmt"
	"os"
	"path/filepath"

	"code.byted.org/gopkg/logs"
	"code.byted.org/kite/kitware"
)

const (
	DATABUS_RPC_PREFIX = "webarch.loanrpc."
	DATABUS_APP_PREFIX = "webarch.app."
)

var (
	AccessLogger kitware.TraceLogger
	CallLogger   kitware.TraceLogger
	logger       *logs.Logger

	databusRPCChannel string
	databusAPPChannel string
)

// NewAccessLogger return a logger for logging server's access log
func NewAccessLogger(logDir string, serviceName string, useScribe bool) *logs.Logger {
	filename := filepath.Join(logDir, serviceName+".access.log")
	category := serviceName + "_access"
	return newFrameLogger(filename, useScribe, category)
}

// NewCallLogger return a logger for logging remote call log
func NewCallLogger(logDir string, serviceName string, useScribe bool) *logs.Logger {
	filename := filepath.Join(logDir, serviceName+".call.log")
	category := serviceName + "_call"
	return newFrameLogger(filename, useScribe, category)
}

func newFrameLogger(file string, useScribe bool, category string) *logs.Logger {
	logger := logs.NewLogger(1024)
	logger.SetLevel(logs.LevelTrace)
	logger.SetCallDepth(3)

	fileProvider := logs.NewFileProvider(file, logs.DayDur, 0)
	fileProvider.SetLevel(logs.LevelTrace)
	if err := logger.AddProvider(fileProvider); err != nil {
		fmt.Fprintf(os.Stderr, "Add file provider error: %s\n", err)
		return nil
	}

	if DatabusLog && databusRPCChannel != "" { // TODO  serviceName, useScribe也可用全局变量ScribeLog来判断， 而减少函数的参数
		databusProvider := logs.NewDatabusProviderWithChannel(DATABUS_RPC_PREFIX+ServiceName, databusRPCChannel) // 此处为RPC log类型
		databusProvider.SetLevel(logs.LevelTrace)
		if err := logger.AddProvider(databusProvider); err != nil {
			fmt.Fprintf(os.Stderr, "Add databus provider error: %s\n", err)
			return nil
		}
	}

	logger.StartLogger()
	return logger
}

func InitLog(level int) {
	if logger == nil {
		logger = logs.NewLogger(1024)
	}
	logger.SetLevel(level)
	logger.SetCallDepth(3)
	logs.InitLogger(logger)
}

func InitConsole() {
	if logger == nil {
		logger = logs.NewLogger(1024)
	}
	consoleProvider := logs.NewConsoleProvider()
	consoleProvider.SetLevel(logs.LevelTrace)
	if err := logger.AddProvider(consoleProvider); err != nil {
		fmt.Fprintf(os.Stderr, "AddProvider consoleProvider error: %s\n", err)
	}
}

func InitFile(level int, filename string, dur logs.SegDuration, size int64) {
	if logger == nil {
		logger = logs.NewLogger(1024)
	}
	fileProvider := logs.NewFileProvider(filename, dur, size)
	fileProvider.SetLevel(level)
	if err := logger.AddProvider(fileProvider); err != nil {
		fmt.Fprintf(os.Stderr, "AddProvider fileProvider error: %s\n", err)
	}
}

func InitDatabus(level int) {
	if logger == nil {
		logger = logs.NewLogger(1024)
	}
	if DatabusLog && databusAPPChannel != "" {
		databusProvider := logs.NewDatabusProviderWithChannel(DATABUS_APP_PREFIX+ServiceName, databusAPPChannel) // 此处为APP log类型
		databusProvider.SetLevel(level)
		if err := logger.AddProvider(databusProvider); err != nil {
			fmt.Fprintf(os.Stderr, "Add databus provider error: %s\n", err)
		}
	}
}
