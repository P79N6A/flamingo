package logs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"code.byted.org/gopkg/logfmt"
)

const (
	LevelTrace = iota
	LevelDebug
	LevelInfo
	LevelNotice
	LevelWarn
	LevelError
	LevelFatal
)

var (
	levelMap = []string{
		"Trace",
		"Debug",
		"Info",
		"Notice",
		"Warn",
		"Error",
		"Fatal",
	}

	levelBytes = [][]byte{
		[]byte("Trace"),
		[]byte("Debug"),
		[]byte("Info"),
		[]byte("Notice"),
		[]byte("Warn"),
		[]byte("Error"),
		[]byte("Fatal"),
	}
)

type LogMsg struct {
	msg   string
	level int
}

type Logger struct {
	callDepth int // callDepth <= 0 will not print file number info.

	isRunning int32
	level     int
	buf       chan *LogMsg
	flush     chan *sync.WaitGroup
	providers []LogProvider

	wg   sync.WaitGroup
	stop chan struct{}
}

// NewLogger make default level is debug, default callDepth is 2, default provider is console.
func NewLogger(bufLen int) *Logger {
	return &Logger{
		level:     LevelDebug,
		buf:       make(chan *LogMsg, bufLen),
		stop:      make(chan struct{}),
		flush:     make(chan *sync.WaitGroup),
		callDepth: 2,
		providers: nil,
	}
}

// 日志输出到屏幕，通常用于Debug模式
func NewConsoleLogger() *Logger {
	logger := NewLogger(1024)
	consoleProvider := NewConsoleProvider()
	consoleProvider.Init()
	logger.AddProvider(consoleProvider)
	return logger
}

func (l *Logger) AddProvider(p LogProvider) error {
	if err := p.Init(); err != nil {
		return err
	}
	l.providers = append(l.providers, p)
	return nil
}

func (l *Logger) SetLevel(level int) {
	l.level = level
}

// DisableCallDepth will not print file numbers.
func (l *Logger) DisableCallDepth() {
	l.callDepth = 0
}

func (l *Logger) SetCallDepth(depth int) {
	l.callDepth = depth
}

// TODO 这里使用通道注册的方式可能会更好，WriteMsg方法可能会造成阻塞
func (l *Logger) StartLogger() {
	if !atomic.CompareAndSwapInt32(&l.isRunning, 0, 1) {
		return
	}
	if l.providers == nil {
		fmt.Fprintln(os.Stderr, "logger's providers is nil.")
		return
	}
	l.wg.Add(1)
	go func() {
		defer func() {
			atomic.StoreInt32(&l.isRunning, 0)

			l.cleanBuf()
			for _, provider := range l.providers {
				provider.Flush()
				provider.Destroy()
			}

			l.wg.Done()
		}()
		for {
			select {
			case logMsg, ok := <-l.buf:
				if !ok {
					fmt.Fprintln(os.Stderr, "buf channel has been closed.")
					return
				}
				for _, provider := range l.providers {
					provider.WriteMsg(logMsg.msg, logMsg.level)
				}
			case wg := <-l.flush:
				l.cleanBuf()
				for _, provider := range l.providers {
					provider.Flush()
				}
				wg.Done()
			case <-l.stop:
				return
			}
		}
	}()
}

func (l *Logger) cleanBuf() {
	for {
		select {
		case msg := <-l.buf:
			for _, provider := range l.providers {
				provider.WriteMsg(msg.msg, msg.level)
			}
		default:
			return
		}
	}
}

func (l *Logger) Stop() {
	if !atomic.CompareAndSwapInt32(&l.isRunning, 1, 0) {
		return
	}
	close(l.stop)
	l.wg.Wait()
}

func (l *Logger) Fatal(format string, v ...interface{}) {
	l.writeMsg(LevelFatal, format, v...)
}

func (l *Logger) Error(format string, v ...interface{}) {
	l.writeMsg(LevelError, format, v...)
}

func (l *Logger) Warn(format string, v ...interface{}) {
	l.writeMsg(LevelWarn, format, v...)
}

func (l *Logger) Notice(format string, v ...interface{}) {
	l.writeMsg(LevelNotice, format, v...)
}

func (l *Logger) Info(format string, v ...interface{}) {
	l.writeMsg(LevelInfo, format, v...)
}

func (l *Logger) Debug(format string, v ...interface{}) {
	l.writeMsg(LevelDebug, format, v...)
}

func (l *Logger) Trace(format string, v ...interface{}) {
	l.writeMsg(LevelTrace, format, v...)
}

func (l *Logger) prefix(level int, ip, psm, logid string, w io.Writer) {
	w.Write(levelBytes[level])
	w.Write([]byte{' '})
	tmArray := timeDate(time.Now())
	w.Write(tmArray[:])
	w.Write([]byte{' '})
	if l.callDepth > 0 {
		_, file, line, ok := runtime.Caller(l.callDepth + 1)
		if !ok {
			file = "???"
			line = 0
		}
		_, _ = file, line
		w.Write([]byte(filepath.Base(file)))
		w.Write([]byte{':'})
		w.Write([]byte(strconv.Itoa(line)))
		w.Write([]byte{' '})
	}
	w.Write([]byte(ip))
	w.Write([]byte{' '})
	w.Write([]byte(psm))
	w.Write([]byte{' '})
	w.Write([]byte(logid))
	w.Write([]byte{' '})
}

func (l *Logger) writeMsg(level int, format string, v ...interface{}) {
	if level < l.level {
		return
	}
	if atomic.LoadInt32(&l.isRunning) == 0 {
		return
	}

	doMetrics(level)

	msg := fmt.Sprintf(format, v...)
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}

	// use timeDate is 10x faster than t.Format
	dateVal := timeDate(time.Now())
	now := string(dateVal[:])

	// 这里使用 + 进行字符串连接，是因为相对于使用Sprintf, 加号的操作性能要高出30倍左右
	prefix := levelMap[level] + " " + now + " "
	if l.callDepth > 0 {
		_, file, line, ok := runtime.Caller(l.callDepth)
		if !ok {
			file = "???"
			line = 0
		}
		file = filepath.Base(file)
		prefix += file + ":" + strconv.Itoa(line) + " "
	}
	// var buffer bytes.Buffer
	// l.prefix(level, &buffer)
	// prefix := buffer.String()
	msg = prefix + msg
	select {
	case l.buf <- &LogMsg{msg: msg, level: level}:
	default:
	}
}

var logEncoderPool = sync.Pool{
	New: func() interface{} {
		var enc LogEncoder
		enc.Encoder = logfmt.NewEncoder(&enc.buf)
		return &enc
	},
}

// WriteKvs write logfmt style logs.
func (l *Logger) WriteKvs(level int, ip, psm, logid string, kvs ...interface{}) {
	if atomic.LoadInt32(&l.isRunning) == 0 {
		return
	}

	enc := logEncoderPool.Get().(*LogEncoder)
	enc.Reset()
	defer logEncoderPool.Put(enc)

	l.prefix(level, ip, psm, logid, &enc.buf)

	if err := enc.EncodeKeyvals(kvs...); err != nil {
		return
	}
	if err := enc.EndRecord(); err != nil {
		return
	}
	msg := enc.buf.String()
	select {
	case l.buf <- &LogMsg{msg: msg, level: level}:
	default:
	}
}

// Flush 将buf中的日志数据一次性写入到各个provider中，期间新的写入到buf的日志会被丢失
func (l *Logger) Flush() {
	if atomic.LoadInt32(&l.isRunning) == 0 {
		return
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	select {
	case l.flush <- wg:
		wg.Wait()
	case <-time.After(time.Second):
		return // busy ?
	}
}
