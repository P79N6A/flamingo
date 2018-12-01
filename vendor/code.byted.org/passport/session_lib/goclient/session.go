package goclient

import (
	"context"
	"regexp"
	"time"

	logger "code.byted.org/gopkg/logs"
	"github.com/bitly/go-simplejson"
)

type Session struct {
	SessionKey string
	Did        int64
	Iid        int64
	Aid        int32
	Url        string
	UidKey     string
	ctx        context.Context

	// 使用simplejson来替换结构体的使用
	sessionDataDict *simplejson.Json

	// 记录session状态数据
	loaded bool  // session数据是否加载
	error  error // 记录当前处理错误
}

var SESSION_PATTERN *regexp.Regexp = regexp.MustCompile(`^[0-9a-f]{32}$`)

func NewSessionObj(sessionKey string, did, iid int64, aid int32, url string, uidKey string, c context.Context) *Session {
	session := &Session{
		Did:    did,
		Iid:    iid,
		Aid:    aid,
		Url:    url,
		UidKey: uidKey,
		ctx:    c,
		loaded: false,
		error:  nil,
	}
	// 判断sessionkey是否合法
	if SESSION_PATTERN.MatchString(sessionKey) {
		session.SessionKey = sessionKey
		// 监控session key和cookie uid key
		tagKV := map[string]string{
			"from": "goclient",
		}
		EmitCounter("has_session_ley", 1, tagKV)
		if len(uidKey) > 0 {
			EmitCounter("has_uid_ley", 1, tagKV)
		}
	} else {
		// 监控sessionkey非法的监控
		logger.Warn("invalid session key %v", sessionKey)
		EmitCounter(METRICS_INVALID_SESSION_KEY, 1, nil)
		session.SessionKey = ""
	}
	return session
}

func (session *Session) load() {
	// session对象单协程使用，没有数据一致性问题, 无需加锁
	if session.loaded {
		return
	}
	// sessionkey为空，则不查询
	if len(session.SessionKey) == 0 {
		// python 在这里是sessaion_data 设置空{}
		session.sessionDataDict = simplejson.New()
		session.loaded = true
		return
	}

	// 分别尝试读session和cookie
	var sessionErr error
	session.sessionDataDict, sessionErr = sessionServiceBackend.Load(session.ctx, session.SessionKey)
	cookieDict, cookieErr := retrieveCookieDict(session)
	// 监控
	monitorCookieUid(session.sessionDataDict, sessionErr, cookieDict, cookieErr)
	// 设置sessionData
	if sessionErr != nil {
		if cookieErr != nil {
			// session和cookie都失败，之后通过重试reload
			logger.Error("remote get session err %v", sessionErr)
			session.error = SYSTEM_ERROR
			return
		}
		// session失败，cookie成功，用cookie补偿
		session.sessionDataDict = cookieDict
		tagKV := map[string]string{
			"from": "goclient",
		}
		EmitCounter(METRICS_USE_COOKIE_UID, 1, tagKV)
	}

	// 加载成功
	session.loaded = true
	session.error = nil
	if session.sessionDataDict == nil {
		session.sessionDataDict = simplejson.New()
	}
	ttl := session.GetExpireAge()
	if ttl <= 0 {
		session.sessionDataDict = simplejson.New()
	}
}

func retrieveCookieDict(session *Session) (*simplejson.Json, error) {
	userId, err := DecryptUid(session.SessionKey, session.UidKey)
	if err != nil {
		return nil, err
	}
	result := simplejson.New()
	result.Set(KEY_USER_ID, userId)
	result.Set(KEY_DEADLINE, time.Now().Unix()+MIN_SESSION_AGE)
	return result, nil
}

func monitorCookieUid(sessionDict *simplejson.Json, sessionErr error, cookieDict *simplejson.Json, cookieErr error) {
	if sessionErr != nil || cookieErr != nil {
		return
	}
	cookieUid, _ := cookieDict.CheckGet(KEY_USER_ID)
	sessionUid, ok := sessionDict.CheckGet(KEY_USER_ID)
	if !ok {
		return
	}
	eq := "True"
	logger.Info("Check |%v| & |%v|\n", cookieUid, sessionUid)
	if cookieUid != sessionUid {
		eq = "False"
	}
	tagKV := map[string]string{
		"from": "goclient",
		"eq":   eq,
	}
	EmitCounter(METRICS_CMP_COOKIE_UID, 1, tagKV)
}

func (session *Session) GetExpireAge() int64 {
	expireAge := int64(0)
	klJson, err := session.Get(KEY_DEADLINE)
	if klJson == nil || err != nil {
		return expireAge
	}
	deadLine, _ := klJson.Int64()
	if deadLine != 0 {
		expireAge = deadLine - time.Now().Unix()
	}
	return expireAge
}

func (session *Session) Get(key string) (*simplejson.Json, error) {
	if session.sessionDataDict == nil {
		session.load()
	}
	if session.error != nil {
		return simplejson.New(), session.error
	}
	value, exist := session.sessionDataDict.CheckGet(key)
	if !exist {
		return nil, nil
	}
	return value, nil
}

func (session *Session) IsLogin() (bool, error) {
	var userId, err = session.GetUserId()
	if err != nil {
		return false, err
	}
	return userId != 0, nil
}

func (session *Session) GetUserId() (int64, error) {
	if session.sessionDataDict == nil {
		session.load()
	}
	if session.error != nil {
		return 0, session.error
	}
	var userIdJson, err = session.Get(KEY_USER_ID)
	if err != nil {
		return 0, err
	}
	if userIdJson != nil {
		return userIdJson.Int64()
	}
	return 0, nil
}
