package auth_client

import (
	"code.byted.org/cloudpush/common/auth"
	"code.byted.org/golf/ssconf"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type PushNotify struct {
	Args    PushArgs    `json:"args"`
	Message PushMessage `json:"message"`
	Extra   NotifyExtra `json:"extra"`
}

type PushArgs struct {
	// 可选值参数ALL_APPS
	App string `json:"app"`
	// 可选值 ['all', 'android', 'ios']
	Os string `json:"os"`
	// 可选值 ['optional', 'normal', 'recommend', 'strong']
	Level string `json:"level"`
	// 可选值 ['all', 'city', 'user_tag', 'no_user_tag', 'device_id', 'device_batch', 'device_list', 'user_id']
	Type string `json:"type"`
	// type为city时必选
	Citys []uint64 `json:"citys"`
	// 如果为true，表示按要发给不在citys里的人
	CityIsExcluded bool `json:"city_is_excluded"`
	// 如果type为user_tag，必选
	Tags []uint64 `json:"tags"`
	// 如果type为device_id，必选
	DeviceID uint64 `json:"device_id"`
	// 如果type为device_batch, 必选
	DeviceBatch []PpsDevice `json:"device_batch"`
	// 如果type为device_list, 必选
	DeviceList []uint64 `json:"device_list"`
	// 如果该字段大于零，表示app_version大于等于该版本才推
	MinAppVersion int `json:"min_app_version"`
	// 如果该字段大于零，表示app_version小于等于该版本才推
	MaxAppVersion int `json:"max_app_version"`
	// 如果为true，表示不限制频率
	NotLimitFrequency bool `json:"not_limit_frequency"`
	// 推送通道个性化: 是否进行声音提醒
	UseSound bool `json:"use_sound"`
	// 推送通道个性化: 是否进行Led灯闪烁
	UseLed bool `json:"use_led"`
	// 推送通道个性化: 是否进行震动提醒
	UseVibrator bool `json:"use_vibrator"`
	// 推送通道个性化: 弹窗类型0,1,2
	AlertType int `json:"alert_type"`
	// 推送通道个性化: 探测包
	IsPing bool `json:"is_ping"`
	//发送的最低活跃时间 为0则不考虑
	MinActiveTimeStamp int64 `json:"min_active_timestamp"`
	//发送的最高活跃时间
	MaxActiveTimeStamp int64 `json:"max_active_timestamp"`

	//根据schema计算出的filterList
	AndroidWhiteFilters []int32 `json:"AndroidWhiteFilters"`
	IOSWhiteFilters     []int32 `json:"IOSWhiteFilters"`
	AndroidMinVer       int     `json:"AndroidMinVer"`
	IOSMinVer           int     `json:"IOSMinVer"`

	//附件的支持
	Attachment string `json:"attachment"`
}

type PpsDevice struct {
	DeviceID uint64 `json:"device_id"`
	// label会在日志中打印出来，方便pps做abtest
	Label string `json:"label"`
}

//这个字段扩充含义,可以用作图片,也可以用作视频,单独解释
type ImageParas struct {
	Type string   `json:"type"`
	Cdns []string `json:"cdns"`
	Uri  string   `json:"uri"`
}

func (p *ImageParas) GetUrl() string {
	if p.Type == "cdn_image" && p.Uri != "" && len(p.Cdns) > 0 {
		return p.Cdns[rand.Int()%len(p.Cdns)] + p.Uri
	}
	//如果是不支持的类型,则返回空
	return ""
}

type PushMessage struct {
	ID uint64 `json:"id"`
	// 可选值 ['unknown', 'article', 'joke', 'user', 'badge']
	Type string `json:"type"`
	// 标题
	Title string `json:"title"`
	// 不能超过2048字节
	Content string `json:"content"`
	// 安卓简介，不能超过MaxPayloadLen字节
	ExtraContent string `json:"extra_content"`
	// 推送文章或段子时，打开对应的文章
	GroupID uint64 `json:"group_id"`
	// 客户端点击通知打开的页面，会被groupid覆盖
	OpenURL string `json:"open_url"`
	// 客户端图片URL
	ImageURL string `json:"image_url"`
	// 客户端图片的类型 (0无图 1大图 2小图)
	ImageType int `json:"image_type"`
	// 客户端锁屏的类型 (0不显示 1大图 2小图 3只文字)
	LockType int `json:"lock_type"`
	// 创建时间，时间戳
	CreateTime uint64 `json:"create_time"`
	// 过期时间，时间戳
	ExpireTime uint64 `json:"expire_time"`
	// 透传给客户端，上游自己定制。
	ExtraStr string `json:"extra_str"`

	//apns专用的category字段,只适用于苹果推送
	ApnsCategory string `json:"apns_category"`

	//对消息打的tag,对于那些不会记录message_id 对应消息信息的调用方,传这个tag可以便于做一些统计
	MsgTag string `json:"msg_tag"`

	// group相关用于hack的数据，由上层传进来，推送发送前，会根据一些hack条件来决定是否发送
	// 比如头条里如果是专题，则只能推头条3.3以上版本，如果带视频，不能推os_api==19的机型
	IsSubject bool `json:"is_subject"`
	HasVideo  bool `json:"has_video"`
}

type NotifyExtra struct {
	IosPayload     string
	AndroidPayload string
	PushPayload    string
	XiaomiPayload  string
	MipushPayload  string
	UmengPayload   string
	HuaweiPayload  string
	AliPayload     string
	MsgType        int
	IsStrong       int
	PushType       int
	Expire         uint64
	//推送调用PushService时间
	AccessTime uint64 `json:"access_time"`
	//推送Rule被PushWorker处理时间
	FetcherTime uint64 `json:"fetcher_time"`
	//上下游都使用的ReqID 用于标识并追踪一次提交到open_service的请求
	ReqID string `json:"reqid"`
	//上下游都使用的JobID，用于聚合任务
	JobID string `json:"jobid"`
}

type OpenServiceResponse struct {
	Message  string `json:"message"`
	OldReqID uint64 `json:"request_id"`
	ReqID    string `json:"reqid"`
}

var serverList, _ = ssconf.GetServerList("/opt/tiger/ss_conf/ss/cloudpush_openservice.conf", "openservice_hosts")
var urlPath = "/pushservice/push_msg/"

func PushMsg(notify *PushNotify, ak string, sk string, timeout time.Duration) (*OpenServiceResponse, error) {
	data, err := json.Marshal(notify)
	if err != nil {
		return nil, fmt.Errorf("fail to marshal notify:%v", notify)
	}
	dataStr := string(data)

	index := rand.Int() % len(serverList)

	for i := 0; i < len(serverList); i++ {
		index = (index + 1) % len(serverList)
		host := serverList[index]
		requrl := "http://" + host + urlPath
		req, err := http.NewRequest("POST", requrl, strings.NewReader(dataStr))
		if err != nil {
			return nil, fmt.Errorf("fail to new request for %v", notify)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", auth.SimpleSign(ak, sk, data))
		client := &http.Client{
			Timeout: timeout,
		}
		res, err := client.Do(req)
		if err != nil {
			//client.Do 报的错误类型都是url.Error,所以只能根据报错信息区分是继续尝试下一台服务器,还是直接返回错误
			if _, ok := err.(*url.Error); ok {
				if strings.Contains(err.Error(), "request canceled while waiting for connection") || strings.Contains(err.Error(), "request canceled (Client.Timeout exceeded while awaiting headers)") {
					return nil, err
				} else {
					// 如果是url错误,并且不是超时,则重试下一台
					continue
				}
			} else {
				return nil, err
			}
		}

		defer res.Body.Close()
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("fail to read response body for %v", notify)
		}

		var result OpenServiceResponse
		err = json.Unmarshal(body, &result)
		if err != nil {
			return nil, fmt.Errorf("fail to unmarshal response for %v, err:%v", notify, err)
		}
		return &result, nil
	}

	return nil, fmt.Errorf("all open service server is unreachable")
}
