package goclient

import "errors"

const (
	DEFAULT_TIME_OUT            = 300
	DEFAULT_CONN_TIME_OUT       = 100
	DEFAULT_CONN_MAX_RETRY_TIME = 500
	SESSION_DATA_KEY            = "session"
	SESSION_COOKIE_NAME         = "sid_tt"

	DEFAULT_SESSION_GET_NAME = "session_key"
	SESSIONID_COOKIE_NAME    = "sessionid"

	PARAM_DID   = "device_id"
	PARAM_IID   = "iid"
	PARAM_AID   = "aid"
	AID_DEFAULT = 13 // news_article
	// use by session
	MIN_SESSION_AGE = 3 * 86400
	// use by django_middleware
	DEFAULT_SESSION_AGE = 30 * 86400

	KEY_DEADLINE           = "dl"
	KEY_USER_ID            = "_spipe_user_id"
	KEY_INSTALL_ID         = "_spipe_install_id"
	KEY_DEVICE_ID          = "_spipe_device_id"
	KEY_PLATFORM           = "_spipe_platform"
	KEY_REMOTE_IP          = "_spipe_remote_ip"
	KEY_USER_TYPE          = "_spipe_user_type"
	KEY_USER_REGISTER_TIME = "_spipe_user_register_time"
	KEY_AID                = "_aid"
	KEY_IP_UPDATE          = "_ip_update"
	KEY_URL_UPDATE         = "_url_update"

	KEY_IS_GRANTED_FROM_SSO      = "_is_granted_from_sso"
	SSO_LOGIN_STATUS_COOKIE_NAME = "sso_login_status"

	// 环境变量
	SESSION_COOKIE_DOMAIN         = "session_cookie_domain"
	DEFAULT_SESSION_COOKIE_DOMAIN = ".snssdk.com"

	SESSION_COOKIE_PATH         = "session.cookie.path"
	DEFAULT_SESSION_COOKIE_PATH = "/"

	SESSION_COOKIE_SECURE         = "session.cookie.secure"
	DEFAULT_SESSION_COOKIE_SECURE = false

	// 监控指标
	METRICS_INVALID_SESSION_KEY     = "invalid_session_key"
	METRICS_UID_CHANGED             = "uid_changed"
	METRICS_CLIENT_GET_SUCCESS      = "client_get_success"
	METRICS_CLIENT_GET_FAIL         = "client_get_fail"
	METRICS_CLIENT_GET_EXCEPTION    = "client_get_exception"
	METRICS_CLIENT_UPDATE_SUCCESS   = "client_update_success"
	METRICS_CLIENT_UPDATE_FAIL      = "client_update_fail"
	METRICS_CLIENT_UPDATE_EXCEPTION = "client_update_exception"
	METRICS_SESSION_ENCRYPT_ERROR   = "session_encrypt_error"
	METRICS_SESSION_DECRYPT_ERROR   = "session_decrypt_error"
	METRICS_USE_COOKIE_UID          = "use_cookie_uid"
	METRICS_CMP_COOKIE_UID          = "cmp_cookie_session_uid"
	METRICS_SID_GET                 = "sid_get"
	METRICS_SID_LOSS                = "sid_loss"
	METRICS_SID_FIX                 = "sid_tt_fix"

	UID_TT_NAME = "uid_tt" // 用于客户端反解uid的cookie
)

var (
	SESSION_KEY_INVALID = errors.New("session key invalid")
	SYSTEM_ERROR        = errors.New("system exception")
)
