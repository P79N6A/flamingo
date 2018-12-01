package alarm

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"code.byted.org/gopkg/logs"
)

// curl -H "Content-Type: application/json" -u 8:401F4DAC3F8A4804BE2A2E999DA83CF7 -X POST -d '{"type":"text","content":{"content":"文本类型"}}' https://ee.byted.org/ratak/dingtalk/channels/47/messages/
type DingTalkAlarm struct {
	urlTemplate     string
	contentTemplate string
	client          *http.Client
	hostname        string
	username        string
	password        string
}

func NewDingTalkAlarm(username, password string) *DingTalkAlarm {
	hostname, _ := os.Hostname()
	return &DingTalkAlarm{
		urlTemplate:     "https://ee.byted.org/ratak/dingtalk/channels/:channel_id/messages/",
		contentTemplate: `{"type":"text","content":{"content":":content"}}`,
		client:          &http.Client{},
		hostname:        hostname,
		username:        username,
		password:        password,
	}
}

// SendDingTalkAlarms to multiple channels, only the last error will be returned if error
func (alarm *DingTalkAlarm) SendDingTalkAlarms(channels []string, content string) error {
	if len(channels) == 0 {
		return errors.New("channels is empty.")
	}
	content = strings.Replace(content, "\"", "'", -1)
	content = fmt.Sprintf("From host %s: %s", alarm.hostname, content)
	var err error
	for _, channelId := range channels {
		err = alarm.sendDingTalkAlarm(channelId, content)
	}
	return err
}

func (alarm *DingTalkAlarm) sendDingTalkAlarm(channelId string, content string) error {
	var jsonStr = []byte(strings.Replace(alarm.contentTemplate, ":content", content, 1))
	url := strings.Replace(alarm.urlTemplate, ":channel_id", channelId, 1)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		logs.Error("Send DingTalk alarm to channel %s error, content is %v", channelId, content)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(alarm.username, alarm.password)

	if resp, err := alarm.client.Do(req); err != nil {
		logs.Error("Send DingTalk alarm to channel %s error, content is %v", channelId, content)
		return err
	} else {
		resp.Body.Close()
		return nil
	}
}
