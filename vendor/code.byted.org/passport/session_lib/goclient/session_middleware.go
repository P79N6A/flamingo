package goclient

import (
	logger "code.byted.org/gopkg/logs"
	"context"
	"net/http"
	"strconv"
)

var SessionProcessMiddleware *SessionMiddleware

func Init(config SessionClientConfig) error {
	// 校验caller不能为空
	if len(config.caller) == 0 {
		panic("caller is necessary")
	}
	// 构造中间件, 传入调用来源
	SessionProcessMiddleware = &SessionMiddleware{}
	// 初始化session服务客户端
	return initSessionClient(config)
}

type SessionMiddleware struct {
}

// 解析出来的session数据会放入request的ctx(SESSION_DATA_KEY)
func (*SessionMiddleware) ProcessRequest(req *http.Request) *http.Request {
	// 用于处理2g/3g网络下cookie被截取的问题,在Header头冗余X_SS_COOKIE
	// COOKIE中不存在的，或者和X_SS_COOKIE值不一致的，以X_SS_COOKIE为准
	rectifyCookie(req)

	var sessionKeyValue string = ""
	// 1. 先从cookie中的sessionid读
	sessionKey, err := req.Cookie(SESSIONID_COOKIE_NAME)
	if sessionKey != nil && err != nil {
		sessionKeyValue = sessionKey.Value
	}
	if !isSessionKeyValid(sessionKeyValue) {
		// 2. sessionid读取失败，从GET参数中读
		sessionKeyValue = retrieveFromGet(req)
		if !isSessionKeyValid(sessionKeyValue) {
			// 3. 从sessionid和GET参数中读取均失败时，尝试从sid_tt中读
			sessionKeyValue = retrieveFromSidTT(req)
		}
	}
	url := req.Host + req.URL.Path
	did, err := strconv.ParseInt(req.FormValue(PARAM_DID), 10, 64)
	iid, err := strconv.ParseInt(req.FormValue(PARAM_IID), 10, 64)
	// aid表示不同的app，默认AID_DEFAULT 为头条
	aid, err := strconv.ParseInt(req.FormValue(PARAM_AID), 10, 32)
	if err != nil {
		aid = AID_DEFAULT
	}
	uidCookie, _ := req.Cookie(UID_TT_NAME)
	var uidKey string
	if uidCookie != nil {
		uidKey = uidCookie.Value
	} else {
		uidKey = ""
	}
	session := NewSessionObj(sessionKeyValue, did, iid, int32(aid), url, uidKey, req.Context())
	// 设置session对象type:Session
	ctx := context.WithValue(req.Context(), SESSION_DATA_KEY, session)
	return req.WithContext(ctx)
}

func rectifyCookie(req *http.Request) {
	cookieStr := req.Header.Get("COOKIE")
	ssCookieStr := req.Header.Get("X_SS_COOKIE")
	// ssCookieStr长度大于cookieStr，说明cookie被截断
	if len(ssCookieStr) >= len(cookieStr) {
		// 提取request的cookie
		originCook := map[string]*http.Cookie{}
		for _, ck := range req.Cookies() {
			originCook[ck.Name] = ck
		}
		// 解析ssCookie
		ssCookies := parseCookies(ssCookieStr)
		for k, v := range ssCookies {
			// cookie缺失
			if _, ok := originCook[k]; !ok {
				logger.Warn("USING_SS_COOKIES, %s=%s not in COOKIES, cookie: %s ss_cookie: %s",
					k, v, cookieStr, ssCookieStr)
			} else if originCook[k].Value != v.Value {
				// cookie值不一致
				logger.Warn("INCONSISTENT_SS_COOKIES, COOKIES[%s]=%s != %s, cookie: %s ss_cookie: %s", k, originCook[k], v, cookieStr, ssCookieStr)
			} else {
				continue
			}
			originCook[k] = v
		}
		for _, v := range originCook {
			req.AddCookie(v)
		}
	}
}

func parseCookies(ssCookieStr string) map[string]*http.Cookie {
	header := http.Header{}
	header.Add("Cookie", ssCookieStr)
	request := http.Request{Header: header}
	cookies := map[string]*http.Cookie{}
	for _, ck := range request.Cookies() {
		cookies[ck.Name] = ck
	}
	return cookies
}

func isSessionKeyValid(sessionKeyValue string) bool {
	return SESSION_PATTERN.MatchString(sessionKeyValue)
}

func retrieveFromGet(req *http.Request) string {
	sessionKeyValue := req.FormValue(DEFAULT_SESSION_GET_NAME)
	if isSessionKeyValid(sessionKeyValue) {
		// 从GET补偿成功
		tagKV := map[string]string{
			"from": "goclient",
		}
		EmitCounter(METRICS_SID_GET, 1, tagKV)
		logger.Warn("Unexpected_get_session, fact_sid: %v host: %v path: %v ip: %v cookie: %v "+
			"http_cookie: %v http_ss_cookie: %v request: %v", sessionKeyValue, req.Host, req.URL.Path,
			req.Header.Get("X-FORWARDED-FOR"), req.Cookies(), req.Header.Get("COOKIE"),
			req.Header.Get("X_SS_COOKIE"), req)
		return sessionKeyValue
	}
	return ""
}

func retrieveFromSidTT(req *http.Request) string {
	tagKV := map[string]string{
		"from": "goclient",
	}
	EmitCounter(METRICS_SID_LOSS, 1, tagKV)
	sessionKey, err := req.Cookie(SESSION_COOKIE_NAME)
	if err != nil {
		return ""
	}
	sessionKeyValue := sessionKey.Value
	if isSessionKeyValid(sessionKeyValue) {
		// 从sid_tt中补偿成功
		EmitCounter(METRICS_SID_FIX, 1, tagKV)
		logger.Info("Fix lost sid: %v", sessionKeyValue)
		return sessionKeyValue
	}
	return ""
}

func (sm *SessionMiddleware) ProcessResponse(req *http.Request, rw http.ResponseWriter) {
	// do nothing
}

func ExtractSession(req *http.Request) *Session {
	sessionObj := req.Context().Value(SESSION_DATA_KEY)

	// session不存在，直接返回
	if sessionObj == nil {
		return nil
	}
	session, ok := sessionObj.(*Session)
	if !ok {
		return nil
	}
	return session
}
