package logs

import (
	"fmt"
	"os"

	"code.byted.org/gopkg/context"
	"code.byted.org/gopkg/net2"
)

var (
	defaultLogger  *Logger
	loadServicePSM = os.Getenv("LOAD_SERVICE_PSM")
	localIP        = net2.GetLocalIp()
)

func init() {
	defaultLogger = NewConsoleLogger()
	defaultLogger.StartLogger()
	defaultLogger.SetCallDepth(3)
}

func InitLogger(logger *Logger) {
	defaultLogger.Stop()
	defaultLogger = logger
	defaultLogger.StartLogger()
}

func AddProvider(p LogProvider) {
	defaultLogger.AddProvider(p)
}

func SetLevel(l int) {
	defaultLogger.SetLevel(l)
}

func SetCallDepth(depth int) {
	defaultLogger.SetCallDepth(depth)
}

func Stop() {
	defaultLogger.Stop()
}

func Fatalf(format string, v ...interface{}) {
	if len(loadServicePSM) > 0 {
		format = fmt.Sprintf("%s %s - %s", localIP, loadServicePSM, format)
	}
	defaultLogger.Fatal(format, v...)
}

func Errorf(format string, v ...interface{}) {
	if len(loadServicePSM) > 0 {
		format = fmt.Sprintf("%s %s - %s", localIP, loadServicePSM, format)
	}
	defaultLogger.Error(format, v...)
}

func Warnf(format string, v ...interface{}) {
	if len(loadServicePSM) > 0 {
		format = fmt.Sprintf("%s %s - %s", localIP, loadServicePSM, format)
	}
	defaultLogger.Warn(format, v...)
}

func Noticef(format string, v ...interface{}) {
	if len(loadServicePSM) > 0 {
		format = fmt.Sprintf("%s %s - %s", localIP, loadServicePSM, format)
	}
	defaultLogger.Notice(format, v...)
}

func Infof(format string, v ...interface{}) {
	if len(loadServicePSM) > 0 {
		format = fmt.Sprintf("%s %s - %s", localIP, loadServicePSM, format)
	}
	defaultLogger.Info(format, v...)
}

func Debugf(format string, v ...interface{}) {
	if defaultLogger.level > LevelDebug {
		return
	}
	if len(loadServicePSM) > 0 {
		format = fmt.Sprintf("%s %s - %s", localIP, loadServicePSM, format)
	}
	defaultLogger.Debug(format, v...)
}

func Tracef(format string, v ...interface{}) {
	if len(loadServicePSM) > 0 {
		format = fmt.Sprintf("%s %s - %s", localIP, loadServicePSM, format)
	}
	defaultLogger.Trace(format, v...)
}

func Fatal(format string, v ...interface{}) {
	if len(loadServicePSM) > 0 {
		format = fmt.Sprintf("%s %s - %s", localIP, loadServicePSM, format)
	}
	defaultLogger.Fatal(format, v...)
}

func Error(format string, v ...interface{}) {
	if len(loadServicePSM) > 0 {
		format = fmt.Sprintf("%s %s - %s", localIP, loadServicePSM, format)
	}
	defaultLogger.Error(format, v...)
}

func Warn(format string, v ...interface{}) {
	if len(loadServicePSM) > 0 {
		format = fmt.Sprintf("%s %s - %s", localIP, loadServicePSM, format)
	}
	defaultLogger.Warn(format, v...)
}

func Notice(format string, v ...interface{}) {
	if len(loadServicePSM) > 0 {
		format = fmt.Sprintf("%s %s - %s", localIP, loadServicePSM, format)
	}
	defaultLogger.Notice(format, v...)
}

func Info(format string, v ...interface{}) {
	if len(loadServicePSM) > 0 {
		format = fmt.Sprintf("%s %s - %s", localIP, loadServicePSM, format)
	}
	defaultLogger.Info(format, v...)
}

func Debug(format string, v ...interface{}) {
	if defaultLogger.level > LevelDebug {
		return
	}
	if len(loadServicePSM) > 0 {
		format = fmt.Sprintf("%s %s - %s", localIP, loadServicePSM, format)
	}
	defaultLogger.Debug(format, v...)
}

func Trace(format string, v ...interface{}) {
	if len(loadServicePSM) > 0 {
		format = fmt.Sprintf("%s %s - %s", localIP, loadServicePSM, format)
	}
	defaultLogger.Trace(format, v...)
}

func Flush() {
	defaultLogger.Flush()
}

func GetContext(ctx context.Context) (ip string, psm string, logid string) {
	ip, psm, logid = "-", "-", "-"
	val := ctx.Value("K_LOCALIP")
	if val != nil {
		ip = val.(string)
	}

	val = ctx.Value("K_SNAME")
	if val != nil {
		psm = val.(string)
	}

	val = ctx.Value("K_LOGID")
	if val != nil {
		logid = val.(string)
	}
	return
}

func CtxFatal(ctx context.Context, format string, v ...interface{}) {
	ip, psm, logid := GetContext(ctx)
	format = fmt.Sprintf("%s %s %s %s", ip, psm, logid, format)
	defaultLogger.Fatal(format, v...)
}

func CtxError(ctx context.Context, format string, v ...interface{}) {
	ip, psm, logid := GetContext(ctx)
	format = fmt.Sprintf("%s %s %s %s", ip, psm, logid, format)
	defaultLogger.Error(format, v...)
}

func CtxWarn(ctx context.Context, format string, v ...interface{}) {
	ip, psm, logid := GetContext(ctx)
	format = fmt.Sprintf("%s %s %s %s", ip, psm, logid, format)
	defaultLogger.Warn(format, v...)
}

func CtxNotice(ctx context.Context, format string, v ...interface{}) {
	ip, psm, logid := GetContext(ctx)
	format = fmt.Sprintf("%s %s %s %s", ip, psm, logid, format)
	defaultLogger.Notice(format, v...)
}

func CtxInfo(ctx context.Context, format string, v ...interface{}) {
	ip, psm, logid := GetContext(ctx)
	format = fmt.Sprintf("%s %s %s %s", ip, psm, logid, format)
	defaultLogger.Info(format, v...)
}

func CtxDebug(ctx context.Context, format string, v ...interface{}) {
	ip, psm, logid := GetContext(ctx)
	format = fmt.Sprintf("%s %s %s %s", ip, psm, logid, format)
	defaultLogger.Debug(format, v...)
}

func CtxTrace(ctx context.Context, format string, v ...interface{}) {
	ip, psm, logid := GetContext(ctx)
	format = fmt.Sprintf("%s %s %s %s", ip, psm, logid, format)
	defaultLogger.Trace(format, v...)
}

func CtxFatalKvs(ctx context.Context, kvs ...interface{}) {
	ip, psm, logid := GetContext(ctx)
	defaultLogger.WriteKvs(LevelFatal, ip, psm, logid, kvs...)
}

func CtxErrorKvs(ctx context.Context, kvs ...interface{}) {
	ip, psm, logid := GetContext(ctx)
	defaultLogger.WriteKvs(LevelError, ip, psm, logid, kvs...)
}

func CtxWarnKvs(ctx context.Context, kvs ...interface{}) {
	ip, psm, logid := GetContext(ctx)
	defaultLogger.WriteKvs(LevelWarn, ip, psm, logid, kvs...)
}

func CtxNoticeKvs(ctx context.Context, kvs ...interface{}) {
	ip, psm, logid := GetContext(ctx)
	defaultLogger.WriteKvs(LevelNotice, ip, psm, logid, kvs...)
}

func CtxInfoKvs(ctx context.Context, kvs ...interface{}) {
	ip, psm, logid := GetContext(ctx)
	defaultLogger.WriteKvs(LevelInfo, ip, psm, logid, kvs...)
}

func CtxDebugKvs(ctx context.Context, kvs ...interface{}) {
	ip, psm, logid := GetContext(ctx)
	defaultLogger.WriteKvs(LevelDebug, ip, psm, logid, kvs...)
}

func CtxTraceKvs(ctx context.Context, kvs ...interface{}) {
	ip, psm, logid := GetContext(ctx)
	defaultLogger.WriteKvs(LevelTrace, ip, psm, logid, kvs...)
}

func CtxPushNotice(ctx context.Context, k, v interface{}) {
	ntc := getNotice(ctx)
	if ntc == nil {
		return
	}
	ntc.PushNotice(k, v)
}

func CtxFlushNotice(ctx context.Context) {
	ntc := getNotice(ctx)
	if ntc == nil {
		return
	}
	ip, psm, logid := GetContext(ctx)
	kvs := ntc.KVs()
	defaultLogger.WriteKvs(LevelNotice, ip, psm, logid, kvs...)
}

func NewNoticeCtx(ctx context.Context) context.Context {
	ntc := newNoticeKVs()
	return context.WithValue(ctx, noticeCtxKey, ntc)
}
