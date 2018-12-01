package ginentry

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"code.byted.org/gopkg/logs"
	"code.byted.org/gopkg/metrics"
	"code.byted.org/kite/kitc"
	"code.byted.org/kite/kitutil"
	_ "code.byted.org/whale/ginentry/clients/webarch/whale/antickite"
	"code.byted.org/whale/ginentry/thrift_gen/whale/anticrawl"
	"github.com/gin-gonic/gin"
)

const (
	MetricsPrefix     = "webarch.whale.antic.middleware"
	WhaleThroughKey   = "Whale-Through"
	WhaleForwardedFor = "Whale-Forwarded"
	WhaleRealIP       = "Whale-Realip"
	WhaleScheme       = "Whale-Scheme"
	WhaleMethod       = "Whale-Method"
	WhaleUA           = "Whale-Ua"
	WhaleReferer      = "Whale-Referer"

	ReqHeaderKeyForwarded = "X-Forwarded-For"
	ReqHeaderKeyProtocol  = "X-Forwarded-Protocol"
	ReqHeaderKeyRealIP    = "X-Real-Ip"
	ReqHeaderKeyUA        = "User-Agent"
	ReqHeaderKeyReferer   = "Referer"
)

// Config anti crawl config
type Config struct {
	EnableAntiCrawl     bool
	AntiCrawlPathsWhite []string
	AntiCrawlPathsBlack []string
}

// NewAntiCrawlConfig 创建配置。notToAnticPathPrefix是路径的白名单，默认使用前缀匹配当前请求，
func NewAntiCrawlConfig(notToAnticPathPrefix []string) *Config {
	return &Config{
		EnableAntiCrawl:     true,
		AntiCrawlPathsWhite: notToAnticPathPrefix,
		AntiCrawlPathsBlack: make([]string, 0, 0),
	}
}

// NewAntiCrawlConfigWithBlackPaths 使用白名单和黑名单创建配置。
// param: notToAnticPathPrefix是白名单，前缀匹配到的请求不发送antic.
// param: toAnticPathPrefix是黑名单，只有匹配到的请求同时没有匹配到白名单的发送antic.
func NewAntiCrawlConfigWithBlackPaths(notToAnticPathPrefix []string, toAnticPathPrefix []string) *Config {
	return &Config{
		EnableAntiCrawl:     true,
		AntiCrawlPathsWhite: notToAnticPathPrefix,
		AntiCrawlPathsBlack: toAnticPathPrefix,
	}
}

func doRequestHasBody(headers map[string][]string) bool {
	_, ctok := headers["Content-Type"]
	_, clok := headers["Content-Length"]
	return ctok && clok
}
func getAnticrawlRequest(request *http.Request, emitCounterFunc func(string, int)) (*anticrawl.AnticrawlRequest, bool, error) {
	originHeaders := map[string][]string(request.Header)
	originQuery := map[string][]string(request.URL.Query())

	errored := false
	bodyData := ""
	if doRequestHasBody(originHeaders) {
		body, berr := ioutil.ReadAll(request.Body)
		request.Body.Close()
		if berr == nil {
			bodyData = string(body)
			request.Body = ioutil.NopCloser(bytes.NewBuffer(body))
		} else {
			logs.Error("[whale.antic] Body is missed for read error.")
			emitCounterFunc("req.body.missed", 1)
			errored = true
		}
	}

	domain := request.Host
	path := request.URL.Path
	anticEnv := make(map[string]string)
	anticHeaders := make(map[string]string)

	//set headers
	for hkey, hvals := range originHeaders {
		if len(hvals) > 0 {
			switch hkey {
			case ReqHeaderKeyForwarded:
				anticHeaders[WhaleForwardedFor] = hvals[0]
			case ReqHeaderKeyProtocol:
				anticHeaders[WhaleScheme] = hvals[0]
			case ReqHeaderKeyRealIP:
				anticHeaders[WhaleRealIP] = hvals[0]
			case anticrawl.HeaderMocked:
				anticEnv[anticrawl.HeaderMocked] = hvals[0]
			default:
				anticHeaders[hkey] = hvals[0]
			}
		}
	}

	anticEnv[WhaleThroughKey] = "ginentry"
	anticHeaders[WhaleMethod] = request.Method

	//set queries
	queries := make(map[string]string)
	for qkey, qvals := range originQuery {
		if len(qvals) > 0 {
			queries[qkey] = qvals[0]
		}
	}

	anticReq := anticrawl.NewAnticrawlRequest()
	anticReq.VersionCode = 1000
	anticReq.Domain = domain
	anticReq.Path = path
	anticReq.Headers = anticHeaders
	anticReq.Queries = queries
	anticReq.Body = &bodyData
	anticReq.AnticEnv = anticEnv

	return anticReq, errored, nil
}

func isRequestToWhaleAntic(path string, enableBlack bool, pathTrie *CTrie) bool {
	candidates, _ := pathTrie.GetCandidateLeafs(path)

	//没有命中trie前缀的情况下，如果不开启黑名单，则发送到antic；开启就不发送
	if len(candidates) == 0 {
		return !enableBlack
	}
	//命中情况下，只要有命中白名单，就不过antic。否则，如果未开启黑名单，返回true；如果开启，只要有一个candidate为true则返回true
	hasBlack := false
	for _, cand := range candidates {
		if cand == false {
			return false
		}
		hasBlack = true
	}
	return !enableBlack || hasBlack
}

// AntiCrawl middleware factory
func AntiCrawl(config *Config) gin.HandlerFunc {
	metricsClient := metrics.NewDefaultMetricsClient(MetricsPrefix, true)
	pathTrie := NewCompressedTrie()

	PSM := os.Getenv("LOAD_SERVICE_PSM")
	if len(PSM) == 0 {
		PSM = "webarch.whale.ginentry"
	}

	enableBlack := false

	if config.AntiCrawlPathsBlack != nil {
		cnt := 0
		for _, blackPrefix := range config.AntiCrawlPathsBlack {
			if len(blackPrefix) > 0 {
				pathTrie.Add(blackPrefix, true)
				cnt++
			}
		}
		if cnt > 0 {
			enableBlack = true
		}
	}
	if config.AntiCrawlPathsWhite != nil {
		for _, whitePrefix := range config.AntiCrawlPathsWhite {
			if len(whitePrefix) > 0 {
				pathTrie.Add(whitePrefix, false)
			}
		}
	}

	client, rpcerr := kitc.NewClient("webarch.whale.antickite",
		kitc.WithConnTimeout(5*time.Millisecond),
		kitc.WithTimeout(10*time.Millisecond),
		kitc.WithConnMaxRetryTime(10*time.Millisecond),
		kitc.WithRPCTimeout(10*time.Millisecond),
	)

	if rpcerr != nil {
		logs.Errorf("[whale antic] RPC Client init error, %v", rpcerr)
	}

	return func(ctx *gin.Context) {
		startTime := time.Now()
		path := ctx.Request.URL.Path
		domain := ctx.Request.Host
		tagKv := map[string]string{
			"domain":  domain,
			"through": "ginentry",
		}
		if rpcerr != nil || client == nil {
			tagKv["error"] = "init_err"
			metrics.EmitCounter("req.error", 1, "", tagKv)
			ctx.Next()
			return
		}
		ctx.Request.Header.Set(anticrawl.HeaderDetected, "1")
		ctx.Request.Header.Set(WhaleThroughKey, "ginentry")

		//path没有命中白名单
		if isRequestToWhaleAntic(path, enableBlack, pathTrie) {
			errored := false
			errorCase := ""
			nextToBackend := true

			defer func() {
				metricsClient.EmitCounter("req.count", 1, "", tagKv)
				if r := recover(); r != nil || errored {
					if r != nil {
						errorCase = "panic"
						logs.Errorf("[whale.antic] panic: %v", r)
					}
					tagKv["error"] = errorCase
					metricsClient.EmitCounter("req.error", 1, "", tagKv)
					ctx.Next()
				} else {
					endTime := time.Now().UnixNano()
					duration := (endTime - startTime.UnixNano()) / 1000000
					metricsClient.EmitTimer("req.time", duration, "", tagKv)

					if nextToBackend {
						ctx.Next()
					} else {
						ctx.Abort()
					}
				}
			}()

			anticReq, reqErrored, rerr := getAnticrawlRequest(ctx.Request, func(mkey string, count int) {
				metricsClient.EmitCounter(mkey, count, "", tagKv)
			})
			if reqErrored {
				errored = true
				errorCase = "getreq_err"
			}

			if rerr == nil {
				rpcCtx := context.Background()
				rpcCtx = kitutil.NewCtxWithRPCTimeout(rpcCtx, 10*time.Millisecond)
				rpcCtx = kitutil.NewCtxWithServiceName(rpcCtx, PSM)
				resp, err := client.Call("GetDecision", rpcCtx, anticReq)
				if err == nil {
					ctx.Request.Header.Set(anticrawl.HeaderDetected, "1")

					anticResp, okResp := resp.RealResponse().(*anticrawl.AnticrawlResponse)
					if !okResp {
						errored = true
						nextToBackend = true
					} else {
						decision := anticResp.GetDecision().GetDecision()
						dconf := anticResp.GetDecision().GetDecisionConf()

						switch decision {
						case anticrawl.DecisionPass:
							nextToBackend = true
						case anticrawl.DecisionBlock:
							nextToBackend = false
							ctx.String(403, "")
						case anticrawl.DecisionCustom:
							nextToBackend = true
							ctx.Request.Header.Set(anticrawl.HeaderDecision, decision)
							ctx.Request.Header.Set(anticrawl.HeaderDecisionConf, dconf)
						case anticrawl.DecisionMock:
							if len(dconf) > 0 && (strings.Index(dconf, "http://") == 0 || strings.Index(dconf, "https://") == 0) {
								nextToBackend = false
								ctx.Request.Header.Set(anticrawl.HeaderMocked, "1")
								ctx.Redirect(http.StatusTemporaryRedirect, dconf)
							} else {
								nextToBackend = true
								metricsClient.EmitCounter("req.decision.mock.error", 1, "", tagKv)
							}
						case anticrawl.DecisionCaptcha:
							//TODO not implemented
							nextToBackend = true
						default:
							errored = true
							errorCase = "decision_notexist"
							nextToBackend = true
						}
					}

				} else {
					errStr := err.Error()
					if strings.Contains(errStr, "timeout") {
						metrics.EmitCounter("req.timeout", 1, "", tagKv)
					} else {
						errorCase = "req_error"
						errored = true
					}
				}
			} else {
				errored = true
			}

		} else {
			ctx.Next()
		}
	}
}
