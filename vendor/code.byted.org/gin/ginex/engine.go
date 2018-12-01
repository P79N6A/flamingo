package ginex

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"

	"path/filepath"

	"code.byted.org/gin/ginex/accesslog"
	"code.byted.org/gin/ginex/apimetrics"
	"code.byted.org/gin/ginex/ctx"
	"code.byted.org/gin/ginex/internal"
	"code.byted.org/gin/ginex/throttle"
	"code.byted.org/gopkg/logs"
	"code.byted.org/gopkg/stats"
	"code.byted.org/whale/ginentry"
	"github.com/gin-gonic/gin"
)

type Engine struct {
	*gin.Engine
}

func New() *Engine {
	r := &Engine{
		Engine: gin.New(),
	}
	return r
}

// write to applog and gin.DefaultErrorWriter
type recoverWriter struct{}

func (rw *recoverWriter) Write(p []byte) (int, error) {
	if appLogger != nil {
		appLogger.Error(string(p))
	}
	return gin.DefaultErrorWriter.Write(p)
}

//Default creates a gin Engine with following middlewares attached:
//  - Recovery
//  - Ctx
//  - Access log
//  - Api metrics
//  - Throttle
func Default() *Engine {
	r := New()

	r.Use(gin.RecoveryWithWriter(&recoverWriter{}))
	r.Use(ctx.Ctx())

	r.Use(accesslog.AccessLog(accessLogger))
	if appConfig.EnableAntiCrawl {
		r.Use(ginentry.AntiCrawl(GetAntiCrawlConfig()))
	}
	r.Use(apimetrics.Metrics(PSM()))
	r.Use(throttle.Throttle())
	return r
}

// Run attaches the router to a http.Server and starts listening and serving HTTP requests.
// It also starts a pprof debug server and report framework meta info to bagent
func (engine *Engine) Run(addr ...string) (err error) {
	if len(addr) != 0 {
		logs.Warnf("Addr param will be ignored")
	}
	if err = Register(); err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() {
		logs.Info("Run in %s mode", appConfig.Mode)
		errCh <- engine.Engine.Run(fmt.Sprintf("0.0.0.0:%d", appConfig.ServicePort))
	}()
	startDebugServer()
	reportMetainfo()
	// start report go gc stats
	stats.DoReport(PSM())
	return waitSignal(errCh)
}

// GETEX是GET的扩展版,增加了一个handlerName参数
// 当handler函数被decorator修饰时,直接获取HandleMethod得不到真正的handler名称
// 这种情况下使用-EX函数显示传入
func (engine *Engine) GETEX(relativePath string, handler gin.HandlerFunc, handlerName string) gin.IRoutes {
	internal.SetHandlerName(handler, handlerName)
	return engine.Engine.GET(relativePath, handler)
}
func (engine *Engine) POSTEX(relativePath string, handler gin.HandlerFunc, handlerName string) gin.IRoutes {
	internal.SetHandlerName(handler, handlerName)
	return engine.Engine.POST(relativePath, handler)
}

func (engine *Engine) PUTEX(relativePath string, handler gin.HandlerFunc, handlerName string) gin.IRoutes {
	internal.SetHandlerName(handler, handlerName)
	return engine.Engine.PUT(relativePath, handler)
}
func (engine *Engine) DELETEEX(relativePath string, handler gin.HandlerFunc, handlerName string) gin.IRoutes {
	internal.SetHandlerName(handler, handlerName)
	return engine.Engine.DELETE(relativePath, handler)
}
func (engine *Engine) AnyEX(relativePath string, handler gin.HandlerFunc, handlerName string) gin.IRoutes {
	internal.SetHandlerName(handler, handlerName)
	return engine.Engine.Any(relativePath, handler)
}

// LoadHTMLRootAt recursively load html templates rooted at \templatesRoot
// eg. LoadHTMLRootAt("templates")
func (engine *Engine) LoadHTMLRootAt(templatesRoot string) {
	var files []string
	filepath.Walk(templatesRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logs.Error("Walk templates directory", templatesRoot, err)
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	engine.Engine.LoadHTMLFiles(files...)
}

func waitSignal(errCh <-chan error) error {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)
	defer logs.Stop()
	defer StopRegister()

	for {
		select {
		case sig := <-ch:
			fmt.Printf("Got signal: %s, Exit..\n", sig)
			return errors.New(sig.String())
		case err := <-errCh:
			fmt.Printf("Engine run error: %s, Exit..\n", err)
			return err
		}
	}
}

// Init inits ginex framework. It loads config options from yaml and flags, inits loggers and setup run mode.
// Ginex's other public apis should be called after Init.
func Init() {
	os.Setenv("GODEBUG", fmt.Sprintf("netdns=cgo,%s", os.Getenv("GODEBUG")))
	loadConf()
	initLog()
	gin.SetMode(appConfig.Mode)

	// MY_CPU_LIMIT will be set as limit cpu cores.
	if v := os.Getenv("MY_CPU_LIMIT"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			runtime.GOMAXPROCS(n)
		}
	}
}
