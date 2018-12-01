package nsq

import (
	"log"
)

// Logger define log interface.
type Logger interface {
	Error(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Info(format string, v ...interface{})
	Notice(format string, v ...interface{})
	Debug(format string, v ...interface{})
}

type defaultLogger struct{}

func (dl *defaultLogger) Error(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (dl *defaultLogger) Warn(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (dl *defaultLogger) Info(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (dl *defaultLogger) Notice(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (dl *defaultLogger) Debug(format string, v ...interface{}) {
	log.Printf(format, v...)
}
