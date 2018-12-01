/*
   提供端口供pprof
*/
package kite

import (
	_ "expvar"
	"net/http"
	_ "net/http/pprof"

	"code.byted.org/gopkg/logs"
)

// RegisterDebugHandler add custom http interface and handler
func RegisterDebugHandler(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	http.HandleFunc(pattern, handler)
}

func startDebugServer() {
	if !EnableDebugServer {
		logs.Info("Debug server not enabled.")
		return
	}

	RegisterDebugHandler("/version", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(ServiceVersion))
	})

	go func() {
		logs.Info("Start pprof listen on: %s", DebugServerPort)
		// Use default mux make easy for new url path like trace.
		err := http.ListenAndServe(DebugServerPort, nil)
		if err != nil {
			logs.Noticef("Start debug server failed: %s", err)
		}
	}()
}
