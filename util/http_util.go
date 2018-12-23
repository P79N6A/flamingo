package util

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"code.byted.org/gopkg/logs"
)

func PostWithObjResponse(c context.Context, url string, params interface{}, respObj interface{}) error {
	bytesData, err := json.Marshal(params)
	if err != nil {
		logs.CtxError(c, "http post marshal failed, req:%+v, err:%v", params, err)
		return err
	}
	client := &http.Client{
		Timeout: time.Second * 5,
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(bytesData))
	if err != nil {
		logs.CtxError(c, "http post NewRequest failed, req:%+v, err:%v", params, err)
		return err
	}
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	resp, err := client.Do(req)
	if err != nil {
		logs.CtxError(c, "http post Do failed, req:%+v, err:%v", params, err)
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logs.CtxError(c, "http post read body failed, req:%+v, err:%v", params, err)
		return err
	}
	logs.CtxInfo(c, "post response:%+v", string(body))
	err = json.Unmarshal(body, &respObj)
	if err != nil {
		logs.CtxError(c, "http response unmarshal failed %v", err)
		return err
	}
	return nil
}
