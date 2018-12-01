package kitc

import (
	"fmt"
	"os"

	"code.byted.org/kite/kitware"
)

var logger kitware.TraceLogger

// SetCallLog which logger for logging calling logs
func SetCallLog(lg kitware.TraceLogger) {
	logger = lg
}

// localLogger implement kitware.TraceLogger interface
type localLogger struct{}

func (l *localLogger) Trace(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", v...)
}

func (l *localLogger) Error(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", v...)
}
