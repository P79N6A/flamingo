package goclient

import (
	logger "code.byted.org/gopkg/logs"
	"code.byted.org/kite/kitc"
	"code.byted.org/kite/kitutil"
	_ "code.byted.org/passport/session_lib/goclient/clients/toutiao/passport/session"
	"code.byted.org/passport/session_lib/goclient/thrift_gen/session"
	"context"
	"errors"
	"github.com/bitly/go-simplejson"
	"time"
)

var SessionClient *kitc.KitcClient

func initSessionClient(config SessionClientConfig) error {
	sessionServiceBackend = &SessionServiceBackend{config.caller}
	connTimeoutOP, timeoutOP, connMaxRetryTimeOP := buildConfigOption(config)

	var err error
	SessionClient, err = kitc.NewClient("toutiao.passport.session", connTimeoutOP, timeoutOP, connMaxRetryTimeOP)
	if err != nil {
		logger.Error("Session Client create client error = %v", err)
	}
	return err
}

func buildConfigOption(config SessionClientConfig) (kitc.Option, kitc.Option, kitc.Option) {
	// 未设置，使用默认配置
	finalConnTimeout := config.connTimeout
	if finalConnTimeout <= 0 {
		finalConnTimeout = DEFAULT_CONN_TIME_OUT
	}
	finalTimeout := config.timeout
	if finalTimeout <= 0 {
		finalTimeout = DEFAULT_TIME_OUT
	}
	finalConnMaxRetryTime := config.connMaxRetryTime
	if finalConnMaxRetryTime <= 0 {
		finalConnMaxRetryTime = DEFAULT_CONN_MAX_RETRY_TIME
	}
	connTimeoutOP := kitc.WithConnTimeout(time.Millisecond * time.Duration(finalConnTimeout))
	timeoutOP := kitc.WithTimeout(time.Millisecond * time.Duration(finalTimeout))
	connMaxRetryTimeOP := kitc.WithConnMaxRetryTime(time.Millisecond * time.Duration(finalConnMaxRetryTime))
	return connTimeoutOP, timeoutOP, connMaxRetryTimeOP
}

var sessionServiceBackend *SessionServiceBackend

type SessionServiceBackend struct {
	caller string
}

func (ssb *SessionServiceBackend) Load(ctx context.Context, sessionKey string) (*simplejson.Json, error) {
	req := session.GetRequest{
		SessionKey: sessionKey,
	}
	// 非Kite框架，显式标记下调用来源
	ctx = kitutil.NewCtxWithServiceName(ctx, ssb.caller)
	resp, err := SessionClient.Call("Get", ctx, &req)
	metricsTagKV := map[string]string{"from": ssb.caller}
	if err != nil {
		EmitCounter(METRICS_CLIENT_GET_EXCEPTION, 1, metricsTagKV)
		logger.Info("session key %s load err: %+v", sessionKey, err)
		return nil, err
	}
	rsp, ok := resp.RealResponse().(*session.GetResponse)
	if !ok || rsp.BaseResp.StatusCode != 0 {
		EmitCounter(METRICS_CLIENT_GET_FAIL, 1, metricsTagKV)
		return nil, errors.New("convert GetResponse failed!")
	}
	EmitCounter(METRICS_CLIENT_GET_SUCCESS, 1, metricsTagKV)
	return simplejson.NewJson([]byte(rsp.SessionData))
}
