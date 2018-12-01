package consul

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const ConfSnapshotArchive = "/opt/tiger/dist/consul_conf_snapshot.zip"

var sd *ServiceDiscovery
var rwlock sync.RWMutex

func handleResponse(resp *http.Response, retObj interface{}) (uint64, error) {
	defer resp.Body.Close()
	watchIndexStr := resp.Header.Get("Watch-Index")
	var index uint64
	if watchIndexStr != "" {
		index, _ = strconv.ParseUint(watchIndexStr, 10, 64)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	json.Unmarshal(body, retObj)
	return index, err
}

func postResponse(path string, bodyObj interface{}, retObj interface{}) (uint64, error) {
	if sd == nil {
		var err error
		sd, err = NewServiceDiscovery()
		if err != nil {
			return 0, err
		}
	}
	url := fmt.Sprintf("http://%s:%d/v1%s", sd.AgentHost, sd.AgentPort, path)
	body, err := json.Marshal(bodyObj)
	if err != nil {
		return 0, err
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	index, err := handleResponse(resp, retObj)
	return index, err
}

func getResponse(path string, retObj interface{}) (uint64, error) {
	if sd == nil {
		var err error
		sd, err = NewServiceDiscovery()
		if err != nil {
			return 0, err
		}
	}
	url := fmt.Sprintf("http://%s:%d/v1%s", sd.AgentHost, sd.AgentPort, path)
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	index, err := handleResponse(resp, retObj)
	return index, err
}

func readFileFromArchive(archive string, fileName string) (string, error) {
	r, err := zip.OpenReader(archive)
	if err != nil {
		return "", err
	}
	defer r.Close()
	for _, f := range r.File {
		if f.Name == fileName {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()
			buf := new(bytes.Buffer)
			buf.ReadFrom(rc)
			return buf.String(), nil
		}
	}
	return "", fmt.Errorf("File not found in archive: %s", fileName)
}

func translateLastResort(trans map[string]string, fileName string) (map[string]string, error) {
	confPath, err := filepath.Abs(fileName)
	if err != nil {
		return nil, err
	}
	confPath = strings.Replace(confPath, "/data00", "/opt", -1)
	confPath = strings.Replace(confPath, "/data12", "/opt", -1)
	snapshot, err := readFileFromArchive(ConfSnapshotArchive, fileName[1:])
	if err != nil {
		return nil, err
	}
	result := make(map[string]string)
	lines := strings.Split(snapshot, "\n")
	for _, line := range lines {
		pos := strings.Index(line, " ")
		if pos < 0 {
			continue
		}
		key := line[0:pos]
		value := line[pos+1:]
		if _, ok := trans[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}

func TranslateOneOnHost(name, agentHost string, agentPort int) ([]*Endpoint, error) {
	if agentHost != defaultAgentHost || agentPort != defaultAgentPort {
		rwlock.Lock()
		defer rwlock.Unlock()

		// Despite of the global "sd" whether has been initialized, to rebuild
		// it while there's a difference between arguments and default-values.
		var err error
		sd, err = NewSpecifiedServiceDiscovery(agentHost, agentPort)
		if err != nil {
			return nil, err
		}
	}

	return TranslateOne(name)
}

func TranslateOne(name string) ([]*Endpoint, error) {
	var endpoints []*Endpoint
	_, err := getResponse(fmt.Sprintf("/lookup/name?name=%s", name), &endpoints)
	if err != nil {
		return nil, err
	}
	return endpoints, nil
}

func TranslateEntry(value string) (string, error) {
	var ret string
	_, err := getResponse(fmt.Sprintf("/lookup/uri?uri=%s", url.QueryEscape(value)), &ret)
	if err != nil {
		return "", err
	}
	return ret, nil
}

func WatchMultiple(services []string, index uint64, timeoutSecs int) (map[string][]*Endpoint, uint64, error) {
	var ret map[string][]*Endpoint
	serviceNames := strings.Join(services, ",")
	index, err := getResponse(fmt.Sprintf("/lookup/watch-multiple?name=%s&index=%d&timeout=%d", url.QueryEscape(serviceNames), index, timeoutSecs), &ret)
	if err != nil {
		return nil, 0, err
	}
	return ret, index, nil
}

func TranslateConf(conf map[string]string, fileName string) (map[string]string, error) {
	newConf := make(map[string]string)
	trans := make(map[string]string)
	for key, value := range conf {
		if strings.HasPrefix(value, "consul:") {
			trans[key] = value
		} else {
			newConf[key] = value
		}
	}
	if len(trans) == 0 {
		return conf, nil
	}
	var transResult map[string]string
	_, err := postResponse("/lookup/conf", trans, &transResult)
	if err != nil {
		transResult, err = translateLastResort(trans, fileName)
	}
	if err != nil {
		return nil, err
	}
	for key, value := range transResult {
		newConf[key] = value
	}
	return newConf, nil
}
